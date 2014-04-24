#goose-demo
使用goose库,用最简单的代码实现检索功能.该项目目的在于演示如何使用[goose](https://github.com/getwe/goose),以及在开发goose过程中同步进行的测试工作.
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
框架并发调用ParseDoc接口,传入一个doc,由策略进行分析返回其`外部id`,`term列表`,`Value`,`Data`.这些元素是在goose中的一个doc的全部表示.具体含义可以参加goose文档.
    
##检索策略
todo
##程序入口

    func main() {
    	app := goose.NewGoose()
    	app.SetIndexStrategy(new(StyIndexer))
    	app.SetSearchStrategy(new(StySearcher))
    	app.Run()
    }
