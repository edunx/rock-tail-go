package tail

import (
	"bufio"
	"context"
	"encoding/gob"
	"errors"
	pub "github.com/edunx/public"
	"github.com/fsnotify/fsnotify"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var Offset = make(map[string]int64)

type Config struct {
	enable     string //开关
	name       string //事件名称
	path       string //tail -f 文件路径, 文件名格式: 0 (名字固定,如error.log); 1 (名字随时间变化)
	offsetFile string //保存偏移量的文件, 不存在则创建
	buffer     int    //缓冲区大小 default:4096
}

// 通过Config.path生成,用于获取和更新tail的文件名
type FileName struct {
	name      string // 配置的原始文件名,如access.{YYYY-MM-dd.HH}.log.ts 或 error.log
	layout    string // {YYYY-MM-dd.HH}
	layoutNew string // 格式, 2006-01-02.03 或 ""
	logType   int    // 名字是否固定(0 固定, 1 不固定)
}

type Tail struct {
	C Config

	transport pub.Transport //transport userdata

	FileName *FileName
	File     *os.File          // 当前打开的文件句柄
	Rd       *bufio.Reader     // bufio读取,随着File变化而变化
	Eof      chan bool         // 当前文件是否读完
	Watcher  *fsnotify.Watcher // 监控文件是否被删除

	Status bool // tail 模块状态

	signalChan chan os.Signal
	//ctx        context.Context
	cancel context.CancelFunc
}

// 从保存offset的文件中获取offset状态
func (t *Tail) GetLoadOffset() {

	offsetFile, err := os.OpenFile(t.C.offsetFile, os.O_RDONLY, 0666)
	if err != nil {
		pub.Out.Err("tail module open offset file error: %v", err)
		return
	}

	defer func() {
		err = offsetFile.Close()
		if err != nil {
			pub.Out.Err("tail module close offset file error: %v", err)
			return
		}
	}()

	err = gob.NewDecoder(offsetFile).Decode(&Offset)
	if err != nil {
		pub.Out.Err("gob decode offset file error: %v", err)
		return
	}
}

// 设置当前待读取文件的offset. 并将文件句柄赋给 t.FIle
func (t *Tail) SetFileOffset(file *os.File) error {
	var offset, tailPos, fromPos int64
	var name string
	var err error

	name = file.Name()
	t.GetLoadOffset()
	offset = Offset[name]

	if tailPos, err = file.Seek(0, 2); err != nil {
		pub.Out.Err("get file tail position for [%s] error : %v", file.Name(), err)
	}

	// 如果当前文件尾部的offset小于保存的offset,则从当前文件的头部开始读取
	if tailPos < offset {
		offset = 0
	}

	fromPos, err = file.Seek(offset, 0)
	if err != nil {
		pub.Out.Err("tail get file current position for [%s] error : %v", file.Name(), err)
		return err
	}

	t.File = file
	pub.Out.Info("tail read file [%s] from position %d", file.Name(), fromPos)
	return nil
}

// 保存文件当前offset, map类型,包含文件名和offset
func (t *Tail) SaveFileOffset() error {
	if t.File == nil {
		return errors.New("file stream is nil, get offset error")
	}

	name := t.File.Name()
	// 获取当前位置
	curPosition, err := t.File.Seek(0, 1)
	if err != nil {
		pub.Out.Err("get file current position for [%s] error : %v", name, err)
		return err
	}

	pub.Out.Debug("current file offsetFile info: {Name: %s, offsetFile: %d}", name, curPosition)

	Offset[name] = curPosition

	offsetFile, err1 := os.OpenFile(t.C.offsetFile, os.O_CREATE|os.O_WRONLY, 0666)
	if err1 != nil {
		pub.Out.Err("open offsetFile file [%s] error: %v", t.C.offsetFile, err1)
		return err1
	}

	defer func() {
		err1 = offsetFile.Close()
		if err1 != nil {
			pub.Out.Err("offsetFile file [%s] close error: %v", t.C.offsetFile, err1)
		}
	}()

	err1 = gob.NewEncoder(offsetFile).Encode(Offset)
	if err1 != nil {
		pub.Out.Err("gob encode offsetFile info error: %v", err1)
		return err1
	}

	return nil
}

// 打开文件操作,并初始化bufio reader
// 文件不存在时,循环读取,直到文件出现或者被取消.
func (f *FileName) openFile(ctx context.Context) (*os.File, error) {
	var name = f.name
	var e error
	var fileStream *os.File

	//tk := time.NewTicker(5 * time.Second)
	//defer tk.Stop()

	for {
		//CHECK:
		if f.logType == 1 {
			now := time.Now().Format(f.layoutNew)
			name = strings.Replace(f.name, f.layout, now, 1)
		}

		fileStream, e = os.OpenFile(name, os.O_RDONLY, 0666)
		if fileStream != nil || !os.IsNotExist(e) {
			return fileStream, e
		}

		pub.Out.Err("file [%s] not exist, check after 5 seconds", name)

		select {
		case <-ctx.Done():
			err := errors.New("open file error, no such file, and open canceled")
			pub.Out.Err("%s", err)
			return nil, err
		default:
			time.Sleep(5 * time.Second)
			//goto CHECK
		}
	}
}

// 捕获系统进程信号,当退出时,保存当前读取文件偏移量信息到文件
func (t *Tail) signalCatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			pub.Out.Err("tail signal catch func exit")
			return
		case sig := <-t.signalChan:
			switch sig {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				if err := t.SaveFileOffset(); err != nil {
					pub.Out.Err("save offsetFile for %s error: %v", t.File.Name(), err)
				} else {
					pub.Out.Info("save offsetFile for %s success", t.File.Name())
				}
				os.Exit(0)
			}
		}
	}
}

// 读取文件
func (t *Tail) Handler(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			pub.Out.Err("tail  handler exit")
			return
		default:
			if t.Rd == nil {
				pub.Out.Err("bufIo reader is nil, maybe no file opened, try again")
				time.Sleep(1 * time.Second)
				continue
			}

			line, err := t.Rd.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					t.Eof <- true
					time.Sleep(1 * time.Second)
					continue
				}
				break
			}
			t.Eof <- false

			if len(line) < 1 {
				continue
			}
			// 去除最后的回车
			t.transport.Push(line[:len(line)-1])
		}
	}
}

// 更新文件句柄, bufio reader, Watcher;
// 情景1: 文件读取到结尾EOF,如果当前正在读取的文件名和应当读取的文件名不一致,则更新;
// 情景2: 文件被删除,需要重新打开句柄
func (t *Tail) Update(ctx context.Context) {
	var name = t.FileName.name

CHECK:
	if t.FileName == nil {
		// 一般情况下不会出现nil的情况
		time.Sleep(10 * time.Second)
		goto CHECK
	}

	for {
		select {
		case <-ctx.Done():
			pub.Out.Info("tail file update func exit")
			return

		case <-t.Eof:
			if t.FileName.logType == 1 {
				now := time.Now().Format(t.FileName.layoutNew)
				name = strings.Replace(t.FileName.name, t.FileName.layout, now, 1)
			}

			if t.File.Name() != name {
				pub.Out.Debug("new name: %s, old name: %s", name, t.File.Name())
				t.DoUpdate(ctx)
			}
		case event, ok := <-t.Watcher.Events:
			if !ok {
				continue
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove && event.Name == t.File.Name() {
				pub.Out.Err("file was removed: %s", t.File.Name())
				t.DoUpdate(ctx)
			}
		case err, ok := <-t.Watcher.Errors:
			if !ok {
				continue
			}
			pub.Out.Err("watcher get error: %v", err)
		}
	}
}

// 监控当前文件所在目录是否存在文件删除事件,
// deprecated
func (t *Tail) DirWatcher(ctx context.Context) {

NEW:
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		pub.Out.Err("new watcher error: %v", err)
		goto NEW
	}

	defer func() {
		if err := watcher.Close(); err != nil {
			pub.Out.Err("close watcher error: %s", err)
		}
	}()

ADD:
	dir := filepath.Dir(t.File.Name())
	err = watcher.Add(dir)
	if os.IsNotExist(err) {
		goto ADD
	}

	for {
		select {
		case <-ctx.Done():
			pub.Out.Err("tail file watcher exit")
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove && event.Name == t.File.Name() {
				pub.Out.Err("file was removed: %s", t.File.Name())
				t.DoUpdate(ctx)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			pub.Out.Err("watcher get error: %v", err)
		}
	}
}

func (t *Tail) NewWatcher() {
NEW:
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		pub.Out.Err("new watcher error: %v", err)
		goto NEW
	}
	t.Watcher = watcher
}

func (t *Tail) AddPath() {
ADD:
	dir := filepath.Dir(t.File.Name())
	err := t.Watcher.Add(dir)
	if os.IsNotExist(err) {
		goto ADD
	}
}

// 执行文件句柄和bufio Reader的更新
func (t *Tail) DoUpdate(ctx context.Context) {
	var err error
	var file *os.File

	if err = t.SaveFileOffset(); err != nil {
		pub.Out.Err("func DoUpdate save file offset error: %v", err)
	}

	file, err = t.FileName.openFile(ctx)
	if err != nil {
		pub.Out.Err("open new file [%s] error: %v", t.FileName.name, err)
		return
	}

	if err := t.Watcher.Remove(t.File.Name()); err != nil {
		pub.Out.Err("tail module file watcher remove path error: %v", err)
	}

	if err = t.File.Close(); err != nil {
		pub.Out.Err("close file [%s] error: %v", t.File.Name(), err)
	}

	err = t.SetFileOffset(file)
	if err != nil {
		pub.Out.Err("set new file [%s] offset error: %v", t.FileName.name, err)
	}

	t.AddPath()
	t.Rd.Reset(t.File)
	t.Rd = bufio.NewReaderSize(t.File, t.C.buffer)
}

func (t *Tail) Start() error {
	var err error

	t.signalChan = make(chan os.Signal)
	t.Eof = make(chan bool)
	signal.Notify(t.signalChan)

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	if t.C.enable != "on" {
		pub.Out.Info("tail enable == off")
		return nil
	}

	// 检验和格式化path
	t.FileName, err = CheckTailFile(t.C.path)
	if err != nil {
		return err
	}

	// 文件不存在时,会循环等待其出现,Start()函数会阻塞,通过 Reload 发送取消ctx 结束阻塞
	file, err := t.FileName.openFile(ctx)
	if err != nil {
		return err
	}

	if err := t.SetFileOffset(file); err != nil {
		return err
	}

	pos, _ := t.File.Seek(0, 1)
	pub.Out.Debug("the start position of [%s] is %d", t.File.Name(), pos)

	t.Rd = bufio.NewReaderSize(t.File, t.C.buffer)
	t.Status = true

	//if err := t.transport.Start(); err != nil {
	//	return err
	//}

	t.NewWatcher()
	t.AddPath()

	go t.Handler(ctx)
	go t.Update(ctx)
	go t.signalCatch(ctx)

	return nil
}

func (t *Tail) Close() {

	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}

	if !t.Status {
		err := errors.New("tail is not running")
		pub.Out.Err("tail close skip, cause %v", err)
		return
	}

	if err := t.SaveFileOffset(); err != nil {
		pub.Out.Err("save file [%s] offset error: %v", t.File.Name(), err)
	}

	if err := t.File.Close(); err != nil {
		pub.Out.Err("file [%s] close error: %v", t.File.Name(), err)
	}

	t.transport.Close()

	if err := t.Watcher.Close(); err != nil {
		pub.Out.Err("tail module file watcher close error: %v", err)
	}

	t.Rd = nil
	t.Status = false
}

// Reload 更新配置后需要重新加载
func (t *Tail) Reload() {
	pub.Out.Debug("current tail config: %v", t.C)

	t.Close()

	if err := t.Start(); err != nil {
		pub.Out.Err("tail module restart error: %v", err)
		return
	}

	t.Status = true
	pub.Out.Info("tail module restart success")
}
