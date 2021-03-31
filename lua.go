package tail

import (
	"github.com/edunx/lua"
)

func (t *Tail) ToLightUserData(L *lua.LState) *lua.LightUserData {
	ud := &lua.LightUserData{Value: t}
	return ud
}

func (t *Tail) LReload(L *lua.LState , args *lua.Args) lua.LValue {
	if e := t.Close(); e != nil {
		L.RaiseError("reload tail.%s close fail" , t.Name())
		return lua.LFalse
	}

	if e := t.Start(); e != nil {
		L.RaiseError("relad tail.%s start fail" , t.Name())
		return lua.LFalse

	}
	return lua.LString(t.Name() + " reload succeed")
}

func (t *Tail) LClose(L *lua.LState , args *lua.Args) lua.LValue {
	if e := t.Close();e != nil {
		L.RaiseError("tail.%s close fail" , t.Name())
		return lua.LNil
	}
	return lua.LString(t.Name() + " close succeed")
}

func (t *Tail) LStart(L *lua.LState , args *lua.Args) lua.LValue {
	if e := t.Start(); e != nil {
		L.RaiseError("tail.%s start fail" , t.Name())
		return lua.LNil
	}

	return lua.LString(t.Name() + " start succeed")
}

func (t *Tail) LDebug(L *lua.LState , args *lua.Args) lua.LValue {
	return lua.LNil
}

func (t *Tail) LToJson(L *lua.LState , args *lua.Args) lua.LValue {
	v , _ := t.ToJson()
	return lua.LString(v)
}

func (t *Tail) Index(L *lua.LState , key string) lua.LValue {

	if key == "transport"  { return t.transport.ToLightUserData(L) }
	if key == "reload"     { return lua.NewGFunction(t.LReload)    }
	if key == "close"      { return lua.NewGFunction(t.LClose)     }
	if key == "start"      { return lua.NewGFunction(t.LStart)     }
	if key == "debug"      { return lua.NewGFunction(t.LDebug)     }
	if key == "json"       { return lua.NewGFunction(t.LToJson)    }

	return lua.LNil
}

func injectTail(L *lua.LState , args *lua.Args) lua.LValue {
	tb := args.CheckTable(L , 1)

	tail := &Tail{
		C: Config{
			name:       tb.RawGetString("name").String(),
			enable:     tb.RawGetString("enable").String(),
			path:       tb.RawGetString("path").String(),
			offsetFile: CheckTailOffset(L, tb),
			buffer:     CheckTailBuffer(L, tb),
		},

		transport: lua.CheckIO(L , tb.RawGetString("transport")),
	}

	ud := &lua.LightUserData{ Value: tail }
	return ud
}

func LuaInjectApi(L *lua.LState, parent *lua.UserKV) {
	parent.Set("tail" , lua.NewGFunction(injectTail))
}