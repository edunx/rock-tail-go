package tail

import (
	"github.com/edunx/lua"
	pub "github.com/edunx/rock-public-go"
	tp "github.com/edunx/rock-transport-go"
)

const (
	MT string = "ROCK_TAIL_GO_MT"
)

func CreateTailUserData(L *lua.LState) int {
	tb := L.CheckTable(1)

	mt := L.GetTypeMetatable(MT)
	ud := L.NewUserData()

	tail := &Tail{
		C: Config{
			name:       tb.RawGetString("name").String(),
			enable:     tb.RawGetString("enable").String(),
			path:       tb.RawGetString("path").String(),
			offsetFile: CheckTailOffset(L, tb),
			buffer:     CheckTailBuffer(L, tb),
		},

		transport: tp.CheckTunnelUserDataByTable(L , tb , "transport"),
	}

	ud.Value = tail
	L.SetMetatable(ud, mt)
	L.Push(ud)

	go func() {
		if err := tail.Start(); err != nil {
			pub.Out.Err("tail start err: %v", err)
		} else {
			pub.Out.Debug("tail start success: %v", tail)
		}
	}()

	return 1
}

func LuaInjectApi(L *lua.LState, parent *lua.LTable) {

	mt := L.NewTypeMetatable(MT)

	L.SetField(mt, "__index", L.NewFunction(Get))

	L.SetField(mt, "__newindex", L.NewFunction(Set))

	L.SetField(parent, "tail", L.NewFunction(CreateTailUserData))
}

func Get(L *lua.LState) int {
	self := CheckTailUserData(L, 1)
	name := L.CheckString(2)

	switch name {
	case "transport":
		L.Push(self.transport.ToUserData(L))
	case "reload":
		L.Push(L.NewFunction(self.reloadByLua))
	case "close":
		L.Push(L.NewFunction(self.closeByLua))
	default:
		L.Push(lua.LNil)
	}
	return 1
}

func Set(L *lua.LState) int {
	self := CheckTailUserData(L, 1)
	name := L.CheckString(2)

	switch name {
	case "enable":
		self.C.enable = L.CheckString(3)
	case "name":
		self.C.name = L.CheckString(3)
	case "path":
		self.C.path = L.CheckString(3)
	case "offset_file":
		self.C.offsetFile = L.CheckString(3)
	case "buffer":
		self.C.buffer = L.CheckInt(3)
	case "transport":
		self.transport = tp.CheckTunnelUserData(L , 3)
	}
	return 0
}

func (t *Tail) ToUserData(L *lua.LState) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = t
	mt := L.GetTypeMetatable(MT)
	L.SetMetatable(ud, mt)
	return ud
}

func(self *Tail) reloadByLua(L *lua.LState) int {
	self.Reload()
	return  0
}

func(self *Tail) closeByLua(L *lua.LState) int {
	self.Close()
	return  0
}