package tail

import (
	"fmt"
	"github.com/edunx/lua"
	pub "github.com/edunx/rock-public-go"
	"github.com/spf13/cast"
	"os"
	"regexp"
	"strings"
	"time"
)

func CheckTailUserData(L *lua.LState, idx int) *Tail {
	ud := L.CheckUserData(idx)
	switch self := ud.Value.(type) {
	case *Tail:
		return self
	default:
		L.ArgError(idx, fmt.Sprintf("args #%d must be tail userdata , got %T", idx, self))
		return nil
	}
}

// 解析文件名,如 access.%{YYYY-MM-dd.HH}.log.ts 解析为 access.2020-11-26.18.log.ts,并返回日志名的模式(0 固定, 1 变化)
// 此处匹配方法不够灵活, todo
func ParseFileName(name string) *FileName {
	var logType int
	var layout string
	var layoutNew string

	reg, err := regexp.Compile("{.*}")
	if err != nil {
		pub.Out.Err("reg compile error: %v", err)
		return &FileName{
			name:      name,
			layout:    "",
			layoutNew: "",
			logType:   0,
		}
	}

	res := reg.FindAll([]byte(name), -1)

	// 匹配到一个 {.*}
	if len(res) == 1 {
		layout = string(res[0])
		layoutNew = strings.Replace(layout, "{", "", 1)
		layoutNew = strings.Replace(layoutNew, "YYYY", "2006", 1)
		layoutNew = strings.Replace(layoutNew, "YY", "06", 1)
		layoutNew = strings.Replace(layoutNew, "MM", "01", 1)
		layoutNew = strings.Replace(layoutNew, "dd", "02", 1)
		layoutNew = strings.Replace(layoutNew, "HH", "15", 1)
		layoutNew = strings.Replace(layoutNew, "mm", "04", 1)
		layoutNew = strings.Replace(layoutNew, "}", "", 1)

		logType = 1
	}

	return &FileName{
		name:      name,
		layout:    layout,
		layoutNew: layoutNew,
		logType:   logType,
	}

}

// 校验fileName: (1)路径为文件夹返回nil,(2)logType为1的文件路径报错,但错误非文件不存在时,返回nil;
// 此处校验是否可跳过?
func CheckFile(fn *FileName) (*FileName, error) {
	now := time.Now().Format(fn.layout)
	name := strings.Replace(fn.name, fn.layout, now, 1)

	stat, err := os.Stat(name)

	if err != nil {
		// 假如文件不存在,直接返回该name,防止在日志文件文件未生成时读取而报错
		if os.IsNotExist(err) {
			return fn, nil
		}

		pub.Out.Err("file got fail ,%v", err)
		return nil, err
	}

	if stat.IsDir() {
		pub.Out.Err("tail path must file , got dir")
		return nil, err
	}

	return fn, nil
}

func CheckTailFile(name string) (*FileName, error) {
	fn := ParseFileName(name)

	return CheckFile(fn)
}

// offset file 自动创建
func CheckTailOffset(L *lua.LState, tb *lua.LTable) string {
	name := tb.RawGetString("offset_file").String()

	stat, err := os.Stat(name)
	if err != nil {
		pub.Out.Err("no such offset file: %s, try to create it", name)

		f, err1 := os.Create(name)
		if err1 != nil {
			pub.Out.Err("try to create offset file error: %v", err1)
			L.RaiseError("get and create offset file error: %v", err)
			return ""
		}

		defer func() {
			_ = f.Close()
		}()

		return name
	}

	if stat.IsDir() {
		L.RaiseError("tail path must file , got dir")
		return ""
	} else {
		return name
	}

}

func CheckTailBuffer(L *lua.LState, tb *lua.LTable) int {
	i := cast.ToInt(tb.RawGetString("buffer").String())
	if i == 0 {
		return 4096
	}
	return i
}
