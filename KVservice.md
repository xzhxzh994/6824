# instruction

在**单机**上构建**kvserver**，保证每个Put在P的情况下最多执行一次，并且是线性的。

You will use this **KV server** to implement a **lock**

Later labs will  replicate a server like this one to handle server crashes.



# KV server:

Client使用**Clerk**来和KVserver交互。

Clerk通过发送RPC请求与Server交互。

RPC有两种：

Put(key,value,version)

Get(key)

对于每个key,  server在**内存的map**中保留 对应的 (**value,version**)  ps：key和value都是string

version number记录了key被写入的时间。

**Put(key, value, version)操作** 将一个特定的key写入/替代 进入map，前提：version number和server的version number一致。

如果version number，一致，那么服务器会**increments**这个version number。

如果不一致：那么服务器return rpc.ErrVersion

创建新的key：version number = 0 ps:server上面就会创建一个version#为1的



**Get(key)**操作：拿到对应Key的value和version。如果对应的Key不存在于server中，那么就return `rpc.ErrNoKey`.



我们需要维护好version#，以便于我们保证当网络不可用时，一次只有一个put操作可以被执行。



完成本实验并通过所有测试后，从调用`Clerk.Get`和`Clerk.Put 的`客户端的角度来看 ，您将拥有一个*线性化的*键/值服务。

也就是说：如果客户端操作不是并发的，那么Get和put都会是线性的。

对于并发操作：返回值和最终的状态会按照某个顺序 排列。**即使这些操作是并发的，从客户端的角度来看，它们的返回值和最终状态应该和这些操作在某个顺序下按顺序执行的结果相同**

对于单一服务器来说，提供线性一致性相对容易。



入门

kvsrv1 `/client.go`实现了一个 Clerk

您需要修改`client.go`和`server.go`

Rpc请求和回复的错误值定义在：rpc包中的``kvsrv1/rpc/rpc.go

```
$ cd ~/6.5840
$ git pull
...
$ cd src/kvsrv1
$ go test -v
=== RUN   TestReliablePut
One client and reliable Put (reliable network)...
    kvsrv_test.go:25: Put err ErrNoKey
...
$
```



## 任务1 （easy）

实现一个方法：当没有dropped messages的时候，我们需要实现Put和Get方法（client.go），以及部署好Put和Get RPC handler (server.go)



测试：

```
$ go test -v -run Reliable
=== RUN   TestReliablePut
One client and reliable Put (reliable network)...
  ... Passed --   0.0  1     5    0
--- PASS: TestReliablePut (0.00s)
=== RUN   TestPutConcurrentReliable
Test: many clients racing to put values to the same key (reliable network)...
info: linearizability check timed out, assuming history is ok
  ... Passed --   3.1  1 90171 90171
--- PASS: TestPutConcurrentReliable (3.07s)
=== RUN   TestMemPutManyClientsReliable
Test: memory use many put clients (reliable network)...
  ... Passed --   9.2  1 100000    0
--- PASS: TestMemPutManyClientsReliable (16.59s)
PASS
ok  	6.5840/kvsrv1	19.681s
```

`每个Passed` 后面的数字分别是实时时间（以秒为单位）、常数 1、发送的 RPC 数量（包括客户端 RPC）以及执行的 key/value 操作数量（`Clerk ``Get`和`Put`调用）。

# 任务2(moderate)

**使用 key/value clerk实现lcok**

在许多分布式系统中，客户端使用kv服务器来协调它们的活动。

比如：zookeeper和Etcd运行使用分布式锁进行协调。类似于Sync.Mutex

他们使用了条件put(conditional put)实现了这个锁。



**your task:**

实现一个基于客户端 `Clerk.Put`和`Clerk.Get`调用的锁

该锁支持两种方法：`Acquire`和`Release`

该锁的规范是，**一次只能有一个客户端成功获取锁**；其他客户端必须等到第一个客户端使用`Release`释放锁

`src/kvsrv1/lock/`中提供了框架代码和测试

需要修改`src/kvsrv1/lock/lock.go `文件

`Acquire`和`Release代码可以通过调用  lk.ck.Put()` 和`lk.ck.Get()`与键/值服务器通信

问题：如果客户端在持有锁时崩溃，该锁将永远不会被释放。

因此常规的优化方法：lease，不过本实验不会crash

测试：

```
$ cd lock 
$ go test -v -run Reliable 
=== RUN TestOneClientReliable
测试：1 个锁定客户端（可靠网络）... 
  ... 通过 -- 2.0 1 974 0 
--- PASS：TestOneClientReliable（2.01s）
=== RUN TestManyClientsReliable
测试：10 个锁定客户端（可靠网络）... 
  ... 通过 -- 2.1 1 83194 0 
--- PASS：TestManyClientsReliable（2.11s）
PASS 
ok 6.5840/kvsrv1/lock 4.120s
```

如果您尚未实现锁，则第一个测试将会成功。

这个练习需要很少的代码，但比上一个练习需要更多的独立思考。

hint

- 您将需要每个锁客户端的唯一标识符；调用`kvtest.RandValue(8)`来生成随机字符串。
- 锁服务应该使用特定的密钥来存储“锁状态”（您必须精确地确定锁状态是什么）。要使用的密钥通过`src/kvsrv1/lock/lock.go` 中`MakeLock`的参数`l`传递。



# 任务3  

会发生消息丢失的KV server



网络可能会**重新排序、延迟或丢弃** RPC 请求和/或回复

为了从丢弃的请求/回复中恢复，Clerk 必须**不断重试每个 RPC**，**直到收到服务器的回复**。

如果网络丢弃了 **RPC 请求消息**，那么**客户端重新发送请求**将解决问题：服务器将接收并执行重新发送的请求。

网络可能会丢弃 **RPC 回复消息**，客户端只会发现没有收到回复。

并且客户端重新发送 RPC 请求，则服务器将收到该请求的两个副本。Get还好，不会修改系统状态，

重新发送具有相同版本号的`Put RPC `是安全的，因为服务器会**根据版本号**有条件地执行   `Put` 操作

如果服务器接收并执行了 `Put` RPC，它将使用`rpc.ErrVersion`响应该 RPC 的重新传输副本，而不是**再次执行 Put 操作**。

一个比较棘手的情况是，如果服务器在 Clerk 重试的 RPC 请求中回复了`rpc.ErrVersion 。`在这种情况下，**Clerk 无法知道服务器是否执行了 Clerk 的`Put`请求**

case1.执行过对应的RPC请求

case2.密钥被更新过，

因此，如果 **Clerk** 收到了重传的 Put RPC 的 `rpc.ErrVersion` ， `Clerk.Put`必须向应用程序返回`rpc.ErrMaybe`而不是`rpc.ErrVersion` ，因为请求可能已经执行。

### 更好的优化：

我们每个操作不会重复发，只发送一次put。这需要维护好每个Clerk的状态。



现在，您应该修改`kvsrv1/client.go`文件，以便在 RPC 请求和回复被丢弃时继续运行。

客户端的`ck.clnt.Call()`返回`true` 表示客户端收到了来自服务器的 RPC 回复；返回`false`表示未收到回复（等待一段时间）

您的 `Clerk`应该不断重新发送 RPC，直到收到回复为止。请记住上面关于`rpc.ErrMaybe`的讨论。

```
向Clerk添加代码，以便在未收到回复时重试。如果您的代码通过了kvsrv1/中的所有测试，则表示您已完成此任务，如下所示：
```

```
$ go test -v 
=== RUN TestReliablePut
一个客户端和可靠的 Put（可靠的网络）... 
  ... Passed -- 0.0 1 5 0 
--- PASS: TestReliablePut (0.00s) 
=== RUN TestPutConcurrentReliable
测试：许多客户端竞相将值放入同一个键（可靠的网络）... 
info: 线性一致性检查超时，假设历史记录正常
  ... Passed -- 3.1 1 106647 106647 
--- PASS: TestPutConcurrentReliable (3.09s) 
=== RUN TestMemPutManyClientsReliable
测试：内存使用许多 put 客户端（可靠的网络）... 
  ... Passed -- 8.0 1 100000 0 
--- PASS: TestMemPutManyClientsReliable (14.61s) 
=== RUN TestUnreliableNet
一个客户端（不可靠的网络）... 
  ...通过 -- 7.6 1 251 208 
--- 通过：TestUnreliableNet (7.60s)
通过
ok 6.5840/kvsrv1 25.319s
```

#### hint:

- 客户端重试之前，应该等待一会儿；您可以使用 go 的`时间`包并调用`time.Sleep(100 * time.Millisecond)`

# 任务4：

### Implementing a lock using key/value clerk and unreliable network

使用kvclerk和不可靠的网络实现分布式锁

修改您的锁实现，使其能够在网络不稳定的情况下与修改后的键/值客户端正常工作。当您的代码通过所有`kvsrv1/lock/` 测试（包括不稳定的测试）时，您就完成了此练习：

```
$ cd lock 
$ go test -v 
=== RUN TestOneClientReliable
测试：1 个锁定客户端（可靠网络）... 
  ... 通过 -- 2.0 1 968 0 
--- 通过：TestOneClientReliable（2.01s）
=== RUN TestManyClientsReliable
测试：10 个锁定客户端（可靠网络）... 
  ... 通过 -- 2.1 1 10789 0 
--- 通过：TestManyClientsReliable（2.12s）
=== RUN TestOneClientUnreliable
测试：1 个锁定客户端（不可靠网络）... 
  ... 通过 -- 2.3 1 70 0 
--- 通过：TestOneClientUnreliable（2.27s）
=== RUN TestManyClientsUnreliable
测试：10 个锁定客户端（不可靠网络）... 
  ... 通过 -- 3.6 1 908 0 
--- 通过： TestManyClientsUnreliable (3.62s) 
PASS 
ok 6.5840/kvsrv1/lock 10.033s
```



# 任务1  实现分布式服务器中的Get和Put操作的原子性

实现RPC调用的PUT和GET功能

call函数已经实现

# 1.实现server.Get函数

接受参数：args：key

reply:value,version,err

1.如果key不存在，err = ErrNokey

2.如果key存在，返回对应的value和version,err = ok

# 2.实现server.Put函数

接收参数：args:key,value,version

返回参数:err

1.检查是否存在这个key,如果不存在:检查version == 0

如果version!=0那么返回err.nokey

如果不存在version==0  创建这个key,对应的value和version+1

2.如果存在这个key

检查version是否相等，

如果相等，则将key对应的VV改为对应的value和version+1

如果不相等，返回errversion

# 3.RPC

实验已经定义好了call函数

只需要实现client.go中的**Put**函数和**Get**函数

使用**clerk.call**来使用rpc

最后只需要直接返回对应的值即可



# 任务2 实现无丢包情况下的分布式锁

实现lock

修改文件：src/kvsrv1/lock/lock.go

lock有两个method :acquire和release

一次只能有一个客户端成功获取锁

`Acquire`和`Release代码可以通过调用  lk.ck.Put()` 和`lk.ck.Get()`与服务器通信

hint

kvtest.RandValue(8)生成8位随机字符串作为唯一标识

应该使用特定的密钥来存储“锁状态”,密钥使用`src/kvsrv1/lock/lock.go` 中`MakeLock`的参数`l`传递

## basicly:

在某个服务器上维护一个值。

当一个client想获得某把锁的时候，它需要把自己的唯一标识符Put到这个值上。

之后其他尝试获得这把锁的的时候，会使用Get检查这个值，如果这个值对应着某个 特殊的string（空闲），那么就说明锁空闲，否则锁



# 1.定义一个锁类型

保存参数：

1.存储锁Key的服务器handle

2.锁标志位的key

3.Make这个locker的client唯一标识符 （一个8bitstring）

# 2.定义makelock

makelock接受参数：维护值的服务器handle和 对应的key

使用指定函数生成一个唯一标识

检查服务器中有无该key,如果没有则创建

返回对应的lock结构体

# 3.定义acquire

一个无限循环

​	acquire方法使用Get获取对应服务器中的key值

​	如果对应key的值为空闲或者为自己的ID，->则尝试PUT自己的ID进去，更新Version。如果Put成功则break

​	如果对应的key不为空闲，则sleep一段时间，再次尝试

# 4.定义Release

使用Get获取对应的key值

如果不是自己的ID，则报错

如果是自己的ID则将其值设定为空闲，退出



# 任务3实现不稳定网络下的Get和Put

对于不稳定的网络下的Get：

我们只需要不断重传  Get请求知道收到确认即可

对于不稳定网络下的Put：

延迟情况：

我们必然会收到两条回复

请求延迟：我们可能重新发送请求，不过第二条请求会被回复ErrVersion，第一条请求得回复会是原本得结果

回复延迟：我们同样会尝试重发请求，与上面同理

丢包 有两种情况：

1.请求丢包  ： 请求未执行，无事发生

2.回复丢包 ：请求已执行，不能接受再执行一次

无论是请求丢包还是回复丢包，客户端得到的丢包的表现就是没有回复。

因此当我们重发过一条消息之后。如果得到的回复是ErrVersion，Clerk不能相信这是真的，因为可能是因为上一条消息送达导致的Version改变，

因此我们重发的消息如果收到了ErrVersion应该改为ErrMaybe，表示有可能是版本不对，也有可能是之前发送的消息生效了。

如果重发的消息得到的回复是：errok，显然说明之前的消息就是丢失了，并且之前的修改也是完全复合规则的

