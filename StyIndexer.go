package main

import (
    "github.com/getwe/goose"
    . "github.com/getwe/goose/utils"
    "github.com/getwe/goose/config"
    "github.com/getwe/scws4go"
	"encoding/json"
    "reflect"
    "runtime"
    log "github.com/getwe/goose/log"
    "strconv"
    "encoding/binary"
)

type oneDocJson struct {
    Title   string
    Docid   string
    Hot     string
    Desc    string
}

// 建库的时候,goose框架建静态库读取文件认为一行是一个doc,动态库一个网络请求是一
// 个doc,这是框架设计.StyIndexer每一个doc是一个json结构,只关注其中4个字段:
//  title:doc的标题,建立索引的字段
//  docid:唯一的外部标志符
//  hot:作为Value,只使用这一个字段,实际情况下可以在Value中存储多个字段
//  desc:附加描述信息,不参与检索
// 这构成了一个最简单的检索元素.
type StyIndexer struct {
    // 共用切词工具
    scws    *scws4go.Scws
}


// 分析一个doc,返回其中的term列表,Value,Data.(必须保证框架可并发调用ParseDoc)
func (this *StyIndexer) ParseDoc(doc interface{},context *goose.StyContext) (
    outId OutIdType,termList []TermInDoc,value Value,data Data,err error) {
    // ParseDoc的功能实现需要注意的是,这个函数是可并发的,使用StyIndexer.*需要注意安全
    defer func() {
        if r := recover();r != nil {
            err = log.Warn(r)
        }
    }()

    // 策略假设每一个doc就是一个[]buf
    realValue := reflect.ValueOf(doc)
    docbuf := realValue.Bytes()

    docJson := oneDocJson{}
    err = json.Unmarshal(docbuf,&docJson)
    if err != nil {
        err = log.Warn(string(docbuf))
        return
    }

    // outid
    idocid,_ := strconv.Atoi(docJson.Docid)
    outId = OutIdType(idocid)

    // 对title进行切词
    segResult,err := this.scws.Segment(docJson.Title)
    if err != nil {
        return
    }

    // 对doc的term进行基础赋权
    // (在一个成熟的检索系统里面,需要一个复杂的子系统来完成工作)
    // 这个测试例子中直接取scws中term的idf
    // 同时,对term也做去重
    termmap := make(map[TermSign]TermWeight)
    for _,term := range segResult {
        tsign := TermSign(StringSignMd5(term.Term))
        tweight := TermWeight(term.Idf * 100)
        if tweight < 1 {
            tweight = 1
        }

        oldwei,ok := termmap[tsign]
        if ok {
            // 取大
            if tweight < oldwei {
                tweight = oldwei
            }
        }
        termmap[tsign] = tweight
    }

    // context.Log输出Info日志不会马上输出,而是由框架最终合并成一行输出
    context.Log.Info("termCount:%d",len(termmap))

    termList = make([]TermInDoc,0,len(termmap))
    for k,v := range termmap {
        termList = append(termList,TermInDoc{
            Sign : k,Weight : v})
    }

    // 从doc中提取需要写入Value的数据
    // 这个策略只使用value的4个字节,写入hot值
    // 合理情况这里应该从配置读取(或者在Init阶段提前读取)Value的长度
    value = NewValue(4)
    hot,_ := strconv.Atoi(docJson.Hot)
    order := binary.BigEndian
    order.PutUint32(value,uint32(hot))

    // 从doc中提取需要写入Data的数据
    // 简单把全部传入的数据当成data返回
    data = NewData(len(docbuf))
    copy(data,docbuf)

    return
}

// 调用一次初始化
func (this *StyIndexer) Init(conf config.Conf) (err error) {

    // scws初始化
    scwsDictPath := conf.String("Strategy.Indexer.Scws.xdbdict")
    scwsRulePath := conf.String("Strategy.Indexer.Scws.rules")
    scwsForkCnt  := runtime.NumCPU()
    this.scws = scws4go.NewScws()
    err = this.scws.SetDict(scwsDictPath, scws4go.SCWS_XDICT_XDB|scws4go.SCWS_XDICT_MEM)
    if err != nil { return }
    err = this.scws.SetRule(scwsRulePath)
    if err != nil { return }
    this.scws.SetCharset("utf8")
    this.scws.SetIgnore(1)
    this.scws.SetMulti(scws4go.SCWS_MULTI_SHORT & scws4go.SCWS_MULTI_DUALITY & scws4go.SCWS_MULTI_ZMAIN)
    err = this.scws.Init(scwsForkCnt)
    if err != nil { return }

    return nil
}

