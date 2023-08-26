配置中的描述字段命名规范：
    1、全部用小写+下划线的方式
    2、币种：btc、eth、doge
    3、合约类型：swap、this_week、next_week、this_quarter、next_quarter

订单生命周期（OKEx）：
订单自行管理自己的生命周期
订单由Trader创建
Trader负责接收统一的ws订单推送，按照订单的ClientId，通过chan发送给各个订单
订单的修改、取消均为自己的接口
订单每隔一段时间未收到任何推送，则自行通过Rest查询自己
非本策略创建的订单：
    A：策略程序不再关心
    B：策略发现后直接取消