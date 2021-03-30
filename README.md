# rock-tail-go
通用的文件读取方法

## 说明
- 磐石系统通用的日志文件读取文件

## 函数说明
- 主要函数为 rock.tail

## rock.tail
- 语法: ud = rock.tail( opt table )
- 参数: lua table的数据结构
```lua
    local ud = rock.tail{
        enable = "on",
        name = "rock.config.access",                    --服务路径名字,格式: 全局变量索引 rock.config.access
        path = "/vdb/logs/access.{YYYY-MM-dd.HH}.log" , -- 读取文件路径 , 支持时间格式
        offset_file = "/vdb/logs/access.offset" ,       -- 上次读取offset
        buffer = 4096,                                  -- 缓存区大小
        transport = lua.IO                              --满足lua.IO的接口
    }
```

- 返回： ud 是个lightuserdata 对象
```lua
    local ud = rock.tail{}
    --返回对象 有下面的方法 
    print(ud.start())
    print(ud.close())
    print(ud.reload())
```
