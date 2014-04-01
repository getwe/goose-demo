// 最简单的策略实现版本,主要用于开发过程中的功能测试.也表明了自定义策略所需要
// 实现必需的和最简单的功能.
//
// goose是一个框架,整个检索系统的一些细节均由策略决定,至少包含以下细节:
//  * 交互网络协议:框架负责数据的网络交互,但其具体数据格式由策略决定.
//  * 索引构建:对一个doc的任何解析均由策略决定,包括转码,分词,Value,Data数据的提取.
//  * Query分析:对检索Query的任何解析由策略决定,包括转码,分词,term重要性打分等.
package main

import (
    "github.com/getwe/goose"
)


func main() {

    app := goose.NewGoose()
    app.SetIndexStrategy(new(StyIndexer))
    app.Run()
}
