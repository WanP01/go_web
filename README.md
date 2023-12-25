# go_web
## go语言构建的简易web框架，主要实现路由匹配功能 (静态，正则，路径参数，通配符 )

###     /9/2    实现map based 路由匹配功能
*******************************************************************
###     /9/27   实现路由树，支持静态，正则，路径参数，通配符四种匹配规则
* ####            路由树的匹配顺序是：
>1. 静态完全匹配 /abc
>2. 正则匹配，形式 /:param_name(reg_expr)
>3. 路径参数匹配：形式 /:param_name
>4. 通配符匹配：/\*
* ####            静态匹配：
>* path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
>* 已经注册了的路由，无法被覆盖。例如 /user/home 注册两次，会冲突
>* 不支持大小写忽略
* ####            通配符匹配：
>* 可支持 a/b/c & a/\* 两个路由同时注册下, a/b/d 匹配（即回溯匹配）
>* a/\*/c 不支持 a/b1/b2/c ,但 a/b/\* 支持 a/b/c1/c2 , 末尾支持多段匹配
* ####            参数匹配：
>* 不能在同一个位置注册不同的参数路由，例如 /user/:id 和 /user/:name 冲突
>* 同名路径参数，在路由匹配的时候，值会被覆盖。例如 /user/:id/abc/:id，那么 /user/123/abc/456 最终 id = 456
* ####            正则匹配：
>* 支持重复路由（即 user/:id[0-9]\* 和 user/:username(\w+) 同时注册），但每次随机选择一项路由执行，不建议
* ####            整体路由匹配：
>* 正则，参数，通配符 不支持重复注册同一节点
>* 不支持并发实现注册（即服务器启动后的注册新路由）
*******************************************************************
###  /10/5  实现Context常用输入输出，AOP模式MiddleWare，template模板解析，文件上传下载 以及 Session 功能
* ####       Context常用输入输出：
>1. RespJson 序列化输出：按照某种特定的格式输出数据，例如 JSON 或者 XML
>2. BindJSON 反序列化输入：将 Body 字节流转换成一个具体的类型(解析body中的数据为json格式,输入到val中)
>3. FormValue 处理表单输入：表单+URL数据按key取值,也是只取 From[key][0]第一个，一般存在表单的情况下取得是表单赋的值
>4. QueryValue 处理查询参数：query URL 查询数据（url ?后面的部分）按key取值，只取第一个 Values[k][0]
>5. PathValue 处理路径参数：查询路由匹配路径的参数（不是url,是url与路由的参数匹配值）
>6. HeaderJson 读取 Header：从 Header 里面读取出来特定的值，并且转化为对应的类型（格式转换可以用StringValue）
* ####       AOP模式MiddleWare：
>1. 限流limiter
>2. prometheus
>3. zipkin/jeager 
>4. 错误/panic处理
* ####       template模板解析：
>1. LoadFromGlob 按照模式导入解析模板
>2. LoadFromFiles 按照文件名导入并解析模板
>3. LoadFromFS  按照文件系统导入并解析模板
* ####       文件上传下载：
>1. FileUpLoader 文件上传
>2. FileDownloader 文件下载
>3. StaticResourceHandler 静态资源请求支持
* ####       Session 功能：
>1. Session session 为结构本体的抽象，支持缓存数据的存取和ID读取
>2. Store 管理 Session，直接面向缓存具体实现（支持redis & go-cache ）
>3. Propagator 面向HTTP，用于包装将缓存对HTTP Header存入/读取/删除/更新的操作
>4. manager 为整合操作，门面设计模式，方便使用者直接调用
*******************************************************************
###  /10/5  自我优化项目
* ####       路由树支持分组&回溯功能：
>1. 回溯优化：
性能：
>   1. ![AddRouter新增回溯功能前BenchMark.png](AddRouter%D0%C2%D4%F6%BB%D8%CB%DD%B9%A6%C4%DC%C7%B0BenchMark.png)
>   2. ![FindRouter新增回溯功能前BenchMark.png](FindRouter%D0%C2%D4%F6%BB%D8%CB%DD%B9%A6%C4%DC%C7%B0BenchMark.png)
   优化后：
>   1. ![AddRouter新增回溯功能后BenchMark.png](AddRouter%D0%C2%D4%F6%BB%D8%CB%DD%B9%A6%C4%DC%BA%F3BenchMark.png)
>   2. ![FindRouter新增回溯功能后BenchMark.png](FindRouter%D0%C2%D4%F6%BB%D8%CB%DD%B9%A6%C4%DC%BA%F3BenchMark.png)
结论： 性能无明显变化
* ####       MiddleWare增加 限流，分组注册等：
>1. ratelimit 实现针对url的访问速度限流
>2. 构建实现分组Group功能，对middleware可用于group上，并实现继承
* ####       自己实现Session内存存储的cache功能：
>1. 构建简易cache缓存，使用LRU/LFU 算法策略实现单机节点的数据淘汰，
>2. 通过锁机制实现缓存并发
>3. 通过slingleflight避免缓存击穿