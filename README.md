#goose-demo
使用goose库,用最简单的代码实现检索功能.该项目目的在于演示如何使用
[goose](https://github.com/getwe/goose),以及在开发goose过程中同步进行的测试工作.
***
#基于goose开发的步骤
##建库策略
建库策略需要实现两个接口:
###初始化接口
    Init(conf config.Conf) (error)
全局调用一次的初始化接口,给策略一次做任何初始化的工作.  
在goose-demo中,策略根据配置初始化切词库,详见示例代码.    
    
###建库分析文档接口
    ParseDoc(doc interface{},context *StyContext) (OutIdType,[]TermInDoc,Value,Data,error)
框架并发调用ParseDoc接口,传入一个doc,由策略进行分析返回其`外部id`,`term列表`,
`Value`,`Data`.这些元素是在goose中的一个doc的全部表示.具体含义可以参加goose文档.
    
##检索策略
建库策略需要实现两个接口:
###初始化接口
    Init(conf config.Conf) (error)
全局调用一次的初始化接口,给策略一次做任何初始化的工作.  
在goose-demo中,策略根据配置初始化切词库,详见示例代码.

###Query分析
    ParseQuery(request []byte,context *StyContext)([]TermInQuery,interface{},error)
框架接受检索请求,不解析请求命令,直接把二进制buffer`request`传递给策略.
策略实现接口,完成解析工作后返回term列表`TermInQuery`.  
整个检索过程需要多个处理流程,ParseQuery第二个返回值由策略自由定制,
后续传递到所有策略接口中.

###文本相关性计算
    CalWeight(queryInfo interface{},inId InIdType,outId OutIdType,
        termInQuery []TermInQuery,termInDoc []TermInDoc,
        termCnt uint32,context *StyContext) (TermWeight,error)
各个参数的含义是:

* queryInfo : ParseQuery策略返回的结构
* inId : 需要打分的doc的内部id
* outId : 需求打分的doc的外部id
* termInQuery : 所有term在query中的打分
* termInDoc : 所有term在doc中的打分
* termCnt : term数量
* Weight : 返回doc的相关性得分

query中的term不一定能命中doc,TermInDoc.Weight == 0表示这种情况.
策略根据term的命中情况计算文本相关性.

###结果过滤
    Filt(queryInfo interface{},list SearchResultList,context *StyContext) (error)
`SearchResultList`是完成计算相关性的结果列表.`Filt`策略定制,
一般用于过滤掉一些不需要的结果.

###调权
    Adjust(queryInfo interface{},list SearchResultList,db ValueReader,context *StyContext) (error)
`CalWeight`计算的是文本相关性,在一个检索系统中仅仅看文本匹配长度一般是不够的,需要额外的加权.  
`ValueReader.ReadValue(InID InIdType) (Value,error)`可以读取指定InID的Value.
Value由建库策略ParseDoc生成,检索策略获取后进行解析并对文本相关性Weight进行调整.

###结果打包
    Response(queryInfo interface{},list SearchResultList,
        db DataBaseReader,response []byte,context *StyContext) (reslen int,err error)
完成调权后,检索结果列表已经确定.`Response`策略queryInfo的信息,
决定要返回的数据写入`response`,同时返回reslen表示返回数据长度.
`DataBaseReader.ReadData(inId InIdType,buf *Data) (error)`读取指定InID的Data.
Data由建库策略ParseDoc生成,检索策略获取后进行解析,根据策略需要写入Response.

##程序入口

    func main() {
    	app := goose.NewGoose()
    	app.SetIndexStrategy(new(StyIndexer))
    	app.SetSearchStrategy(new(StySearcher))
    	app.Run()
    }
