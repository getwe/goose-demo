package main

import (
    . "github.com/getwe/goose"
    . "github.com/getwe/goose/utils"
    . "github.com/getwe/goose/database"
    "github.com/getwe/goose/config"
    "github.com/getwe/scws4go"
	//"encoding/json"
    //"reflect"
    "errors"
    "runtime"
    log "github.com/getwe/goose/log"
    //"strconv"
    //"encoding/binary"
    "sort"

    "code.google.com/p/goprotobuf/proto"
)

// 策略的自定义临时数据
type strategyData struct {
    query   string
    pn      int32
    rn      int32
}

// 检索的时候,goose框架收到一个完整的网络请求便认为是一次检索请求.
// 框架把收到的整个网络包都传给策略,不关心具体的检索协议.
//
// 在这个检索demo中,检索策略以protocolbuf协议解析请求包,并从中获取用户query.
// 具体协议见search.proto
type StySearcher struct {
    scws    *scws4go.Scws
}

// 全局调用一次初始化策略
func (this *StySearcher) Init(conf config.Conf) (err error) {
    // scws初始化
    scwsDictPath := conf.String("Strategy.Searcher.Scws.xdbdict")
    scwsRulePath := conf.String("Strategy.Searcher.Scws.rules")
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

    return
}

// 解析请求
// 返回term列表,一个由策略决定的任意数据,后续接口都会透传
func (this *StySearcher) ParseQuery(request []byte,
    context *StyContext)([]TermInQuery,interface{},error) {

    // 策略在多个接口之间传递的数据
    styData:= &strategyData{}

    // 解析命令
    searchReq := &SearchRequest{}
    err := proto.Unmarshal(request, searchReq)
    if err != nil {
        log.Warn(err)
        return nil,nil,err
    }

    styData.query = searchReq.GetQuery()
    styData.pn = searchReq.GetPn()
    styData.rn = searchReq.GetRn()

    context.Log.Info("query",styData.query)

    // 对query进行切词
    segResult,err := this.scws.Segment(styData.query)
    if err != nil {
        log.Warn(err)
        return nil,nil,err
    }

    termInQ := make([]TermInQuery,0)
    for _,term := range segResult {
        tsign := TermSign(StringSignMd5(term.Term))
        // term重要性:取term长度占比
        tweight := TermWeight( len(term.Term) / len(styData.query) * 100 )

        termInQ = append(termInQ,TermInQuery{
            Sign : tsign,
            Weight : tweight,
            CanOmit : true,
            SkipOffset : true})
    }

    return termInQ,styData,nil
}

// 对一个结果进行打分,确定相关性
// queryInfo    : ParseQuery策略返回的结构
// inId         : 需要打分的doc的内部id
// outId        : 需求打分的doc的外部id
// termInQuery  : 所有term在query中的打分
// termInDoc    : 所有term在doc中的打分
// termCnt      : term数量
// Weight       : 返回doc的相关性得分
// 返回错误当前结果则丢弃
// @NOTE query中的term不一定能命中doc,TermInDoc.Weight == 0表示这种情况
func (this *StySearcher) CalWeight(queryInfo interface{},inId InIdType,
    outId OutIdType,termInQuery []TermInQuery,termInDoc []TermInDoc,
    termCnt uint32,context *StyContext) (TermWeight,error) {

    // 核心相关性打分
    // 最简单demo,把命中doc的得分相加
    var weight TermWeight
    for _,t := range termInDoc {
        weight += t.Weight
    }

    return weight,nil
}

// 对结果拉链进行过滤
func (this *StySearcher) Filt(queryInfo interface{},list SearchResultList,
    context *StyContext) (error) {
    return nil
}

// 结果调权
// 确认最终结果列表排序
func (this *StySearcher) Adjust(queryInfo interface{},list SearchResultList,
    db ValueReader,context *StyContext) (error) {

    // 不调权,直接排序返回
    sort.Sort(list)
    return nil
}

// 构建返回包
func (this *StySearcher) Response(queryInfo interface{},list SearchResultList,
    db DataBaseReader,response []byte,context *StyContext) (err error) {

    styData := queryInfo.(*strategyData)
    if styData == nil {
        return errors.New("StrategyData nil")
    }

    // 分页
    begin := styData.pn * styData.rn
    end := begin + styData.rn
    if end > int32(len(list)) {
        end = int32(len(list))
    }
    relist := list[begin:end]

    searchRes := &SearchResponse{}
    searchRes.Result = make([]*SearchResponseOneRes,len(relist))

    tmpData := NewData()

    for _,e := range relist {
        db.ReadData(e.InId,&tmpData)
        searchRes.Result = append(searchRes.Result,&SearchResponseOneRes{
            Data : tmpData})
    }

    (*searchRes.DispNum) = int32(len(list))
    (*searchRes.RetNum) = int32(len(relist))

    // 进行序列化
    tmpbuf,err := proto.Marshal(searchRes)
    if err != nil {
        return err
    }

    if len(tmpbuf) > cap(response) {
        return errors.New("respone buf too small")
    }

    // 重复了一次内存拷贝!
    copy(response,tmpbuf)

    return nil
}


