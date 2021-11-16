# tomlgen

> 根据toml文件生成对应的结构体以及读取配置文件的方法

## exmaple

config.toml

```toml
title = "TOML Example"
[owner]
name = "Tom Preston-Werner"
dob = 1979-05-27T07:32:00-08:00 # First class dates
```

```shell
tomlgen config.toml
```

stdout

```go
package main

import (
        "github.com/pelletier/go-toml"
        "io/ioutil"
)

type Config struct {
        Title string `toml:"title"`
        Owner Owner  `toml:"owner"`
}
type Owner struct {
        Name string    `toml:"name"`
        Dob  time.Time `toml:"dob"`
}

var Cfg Config

func init() {
        file, err := ioutil.ReadFile("config.toml")
        if err != nil {
                panic(err)
        }
        err = toml.Unmarshal(file, &Cfg)
        if err != nil {
                panic(err)
        }
}
```

更多用法

```shell
# 将结果重定向到config.go文件中
tomlgen -o config.go config.toml

# 指定包名
tomlgen -p test config.toml

# 指定结构体名
tomlgen -n Config config.toml

# 指定变量名
tomlgen -v Cfg config.toml

# 指定生成出的go文件使用的配置文件路径
tomlgen -c ../../config.toml config.toml
```

不生成go文件，而是修改原有的go文件，

```go
package config

import (
	"github.com/pelletier/go-toml"
	"io/ioutil"
)

//go:generate tomlgen -o "config.go" type -n "Config" "../../conf/config.toml"
type Config struct {
}

var Cfg Config

//go:generate tomlgen -o "config.go" init -n "Cfg" "conf/config.toml"
func init() {
}

```