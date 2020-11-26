# GoINI[![PkgGoDev](https://pkg.go.dev/badge/github.com/xiaoqidun/goini)](https://pkg.go.dev/github.com/xiaoqidun/goini)
简单易用的Golang INI配置解析库
# 安装方法
go get -u github.com/xiaoqidun/goini
# 读取配置
## 从文件读取配置
```go
//初始GoINI对象
ini := goini.NewGoINI()
//从文件获取配置
if err := ini.LoadFile("./config.ini"); err != nil {
	log.Println(err)
	return
}
```
## 从字符读取配置
```go
//初始GoINI对象
ini := goini.NewGoINI()
//从字符获取配置
ini.SetData([]byte(""))
```
# 注释方法
goini将;或#开头的行识别为注释信息
# 分区支持
goini将[]（英文中括号）识别为分区
