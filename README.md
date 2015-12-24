# webx

webx是一个强大的Go语言web框架。

[English](https://github.com/coscms/webx/blob/master/README_EN.md)

[![Go Walker](http://gowalker.org/api/v1/badge)](http://gowalker.org/github.com/coscms/webx) [![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/go-webx/webx/trend.png)](https://bitdeli.com/free "Bitdeli Badge")

## 技术支持

QQ群：369240307

## 更新日志
* **v0.5.0** : 
	1. Server支持平滑关闭；
	2. 静态文件和模板文件仅在使用时才缓存（优点：加快服务生效时间和节约内存），支持根据配置自动改变监控目录；
	3. 支持合并静态资源，支持同步更新合并的静态资源；
	4. 改进路由注册方式（现在变得更加智能）；
	5. Server支持限制最大连接数
	6. 新增对action新返回类型：webx.JSON/webx.JSONP/webx.XML/webx.FILE等的支持
	7. app支持绑定子域名
	8. 改进https服务
	9. 其它细微调整
* **v0.4.0** : 
	1. AddTmplVar改为Assign；
	2. AddTmplVars改为MultiAssign；
	3. 日志中增加IP、页面字节大小以及耗时记录（便于查找恶意访问来源）；
	4. 修复bug
* **v0.3.0** : 增加对称加密、XSRF通用接口，更换hook引擎为更加优雅的events引擎
* **v0.2.1** : 自动Binding新增对jquery对象，map和array的支持。
* **v0.2** : 新增 validation 子包，从 [https://github.com/astaxie/beego/tree/master/validation](http://https://github.com/astaxie/beego/tree/master/validation) 拷贝过来。
* **v0.1.2** : 采用 [github.com/coscms/webx/lib/httpsession](http://github.com/coscms/webx/lib/httpsession) 作为session组件，API保持兼容；Action现在必须从*Action继承，这个改变与以前的版本不兼容，必须更改代码；新增两个模板函数{{session "key"}} 和 {{cookie "key"}}；Action新增函数`MapForm`
* **v0.1.1** : App新增AutoAction方法；Action新增Assign方法；Render方法的模版渲染方法中可以通过T混合传入函数和变量，更新了[快速入门](https://github.com/coscms/webx/tree/master/docs/intro.md)。
* **v0.1.0** : 初始版本

## 特性

* 在一个可执行程序中多Server(http,tls,scgi,fcgi)，多App的支持
* 简单好用的路由映射方式
* 静态文件及版本支持，并支持自动加载，默认开启
* 改进的模版支持，并支持自动加载，动态新增模板函数
* session支持
* validation支持

## 安装

在安装之前确认你已经安装了Go语言. Go语言安装请访问 [install instructions](http://golang.org/doc/install.html). 

安装 webx:

    go get github.com/coscms/webx

## Examples

请访问 [examples](https://github.com/coscms/webx/tree/master/examples) folder

## 案例

* [xorm.io](http://xorm.io) - [github.com/go-xorm/website](http://github.com/go-xorm/website)
* [Godaily.org](http://godaily.org) - [github.com/govc/godaily](http://github.com/govc/godaily)

## 文档

[快速入门](https://github.com/coscms/webx/tree/master/docs/intro.md)

源码文档请访问 [GoWalker](http://gowalker.org/github.com/coscms/webx)

## License
BSD License
[http://creativecommons.org/licenses/BSD/](http://creativecommons.org/licenses/BSD/)



