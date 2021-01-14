package tail

import (
	"github.com/edunx/lua"
	"github.com/edunx/public"
)

const (
	MT string = "tail_mt"
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

		transport: CheckTransportByTable(L, tb),
	}

	ud.Value = tail
	L.SetMetatable(ud, mt)
	L.Push(ud)

	go func() {
		if err := tail.Start(); err != nil {
			out.Err("tail start err: %v", err)
		} else {
			out.Debug("tail start success: %v", tail)
		}
	}()

	return 1
}

func LuaInjectApi(L *lua.LState, parent *lua.LTable , output public.Logger) {

	mt := L.NewTypeMetatable(MT)

	L.SetField(mt, "__index", L.NewFunction(Get))

	L.SetField(mt, "__newindex", L.NewFunction(Set))

	L.SetField(parent, "tail", L.NewFunction(CreateTailUserData))

	//注入 out
	out = output
}

func Get(L *lua.LState) int {
	self := CheckTailUserData(L, 1)
	name := L.CheckString(2)

	switch name {
	case "transport":
		L.Push(self.transport.ToUserData(L))
	case "reload":
		L.Push(L.NewFunction(func(L *lua.LState) int {
			self.Reload()
			return 0
		}))
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
		self.transport = public.CheckTransport(L.CheckUserData(3))
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
