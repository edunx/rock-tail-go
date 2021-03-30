package tail

import (
	"bufio"
	"context"
	"github.com/edunx/lua"
	"github.com/fsnotify/fsnotify"
	"os"
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
	lua.Super
	C Config

	transport lua.IO //transport userdata

	FileName *FileName
	File     *os.File          // 当前打开的文件句柄
	Rd       *bufio.Reader     // bufio读取,随着File变化而变化
	Eof      chan bool         // 当前文件是否读完
	Watcher  *fsnotify.Watcher // 监控文件是否被删除
	status   lua.LightUserDataStatus // tail 模块状态
	uptime   time.Time

	signalChan chan os.Signal
	//ctx        context.Context
	cancel context.CancelFunc
	count    int64
}