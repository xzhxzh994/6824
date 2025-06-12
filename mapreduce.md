**你的任务（中等/困难）**

你的任务是实现一个分布式的 MapReduce 系统，由两个程序组成：协调器（coordinator）和工作节点（worker）。系统将只有一个协调器进程，和一个或多个并行执行的工作节点进程。在真实的系统中，工作节点会运行在不同的机器上，但在这个实验中，你将把它们都运行在同一台机器上。工作节点通过 RPC（远程过程调用）与协调器进行通信。

每个工作节点进程会在一个循环中：

1. 向协调器请求任务；(task)
2. 从一个或多个文件中读取任务输入；(filename)
3. 执行该任务；
4. 将任务的输出写入到一个或多个文件；(write)
5. 然后再次向协调器请求新任务。

协调器应该能够检测到如果某个工作节点在合理的时间内（对于本实验，使用10秒）未完成其任务，协调器将把相同的任务分配给另一个工作节点。



任务：

在src/main中给出若干以pg-开头的文件，



如何运行wordcount代码

1.把wc的map函数和reduce函数做成动态库

```shell
$ go build -buildmode=plugin ../mrapps/wc.go
```

2.在main下运行mrcoordinator.go

```
$ rm mr-out*
$ go run mrcoordinator.go pg-*.txt
```

3.开启工作节点,运行若干个工作节点

这里的mrworker实际上调用mr/worker.go中的Worker函数

```
$ go run mrworker.go wc.so=
```

4.之后所有结果将会保存在mr-out-*文件中。





我们为你提供了一个测试脚本 **main/test-mr.sh**。该脚本将检查 wc 和 indexer MapReduce 应用程序在给定 pg-xxx.txt 文件作为输入时是否产生正确的输出。测试还会检查你实现的 Map 和 Reduce 任务是否能并行执行，并且在工作节点崩溃时你的实现是否能正确恢复。



如果什么都没做直接运行，那么脚本会卡住，因为master不会停止,因为其中有一个m.done

```
for m.Done() == false {
		time.Sleep(time.Second)
	}
```

```
$ cd ~/6.5840/src/main
$ bash test-mr.sh
```



测试失败的原因

测试脚本期望在 `mr-out-X` 文件中看到输出，这些文件对应每个 Reduce 任务。然而，由于 `mr/coordinator.go` 和 `mr/worker.go` 的实现是空的，这些文件并没有被创建（或者没有做其他任何操作），因此测试失败。





脚本应该有的输出：

```
$ bash test-mr.sh
*** Starting wc test.
--- wc test: PASS
*** Starting indexer test.
--- indexer test: PASS
*** Starting map parallelism test.
--- map parallelism test: PASS
*** Starting reduce parallelism test.
--- reduce parallelism test: PASS
*** Starting job count test.
--- job count test: PASS
*** Starting early exit test.
--- early exit test: PASS
*** Starting crash test.
--- crash test: PASS
*** PASSED ALL TESTS
$
```



可能出现的错误：

RPC包错误：

```
2019/12/16 13:27:09 rpc.Register: method "Done" has 1 input parameters; needs exactly three
```

这些错误可以忽略。它们是因为协调器在注册 RPC 服务时会检查所有方法是否符合 RPC 的要求（例如，需要三个输入参数）。但是我们知道 `Done` 方法不会通过 RPC 调用，因此可以忽略这些错误。

工作节点与协调器无法连接：

```
2025/02/11 16:21:32 dialing:dial unix /var/tmp/5840-mr-501: connect: connection refused
```

这些错误也可以忽略。它们是因为在协调器退出后，工作节点无法与协调器的 RPC 服务器建立连接。每个测试可能会看到少量这种错误信息，这并不影响测试结果。



# RULES

**1.map阶段**：map产生的**intermediate keys**  放入 **bucket**中。暂时定义bucket为磁盘文件。这些buckets要为**nreduce**个reduce tasks所用，这个nreduce是 mrcoordinator.go中的一个可传入参数，每个 mapper 应该为每个 Reduce 任务创建 `nReduce` 个中间文件，供 Reduce 任务使用。

nreduce可以理解为reduce task的阶段数。

这个nreduce 指的是reduce阶段的任务数量，也就是需要启动多少个reduce worker来并行处理中间结果

**文件修改：**
 你可以修改 `mr/worker.go`、`mr/coordinator.go` 和 `mr/rpc.go` 文件。你可以临时修改其他文件进行测试，但确保你的代码与原始版本兼容，因为我们将使用原始版本来进行测试。



**intermidiate file :**

Worker 应该将中间 Map 输出文件放在当前目录下，**worker** 后续可以**读取这些文件作为 Reduce** 任务的输入。

**Done() 方法：**
 `main/mrcoordinator.go` 文件期望 `mr/coordinator.go` 中实现一个 `Done()` 方法，当 MapReduce 作业完成时返回 `true`，此时 `mrcoordinator.go` 会退出。



**作业完成时退出：**
 当作业完成时，**worker 进程应该退出**。一种简单的实现方式是利用 `call()` 返回值：**如果 worker 无法联系到协调器，说明协调器已经退出（任务完成），此时 worker 也可以终止**。根据你的设计，你可能还会发现**使用一个“请退出”的伪任务很有用**，协调器可以将该任务**分配给 worker**，通知 worker 任务已完成。





**HINT**

**提示**

指导页面提供了一些开发和调试的提示。
 开始的一个好方法是**修改 `mr/worker.go` 中的 `Worker()` 函数，使其向协调器发送一个 RPC 请求**，获取一个任务。然后修改**协调器**，使其**返回一个尚未开始的 Map 任务的文件名**。接着**修改 worker 来读取这个文件并调用应用程序的 Map 函数**，正如在 `mrsequential.go` 中所做的那样。

应用程序的 Map 和 Reduce 函数在运行时**使用 Go 的插件包加载**，从文件中加载，这些文件的名称以 `.so` 结尾。

如果你改变了 `mr/` 目录中的任何文件，通常需要**重新构建你使用的 MapReduce 插件**，像这样：

```
bash


复制编辑
go build -buildmode=plugin ../mrapps/wc.go
```

本实验依赖于所有工作节点共享文件系统。当所有工作节点都在同一台机器上运行时，这没有问题，但**如果工作节点分布在不同的机器上，则需要像 GFS 这样的全局文件系统**。

一个合理的中间文件命名约定是 `mr-X-Y`，其中 X 是 Map 任务的编号，Y 是 Reduce 任务的编号。

worker 实现应将 X 号 Reduce 任务的输出放入 `mr-out-X` 文件中。

worker 的 Map 任务代码需要一种方法来将中间的键值对存储到文件中，以便在 Reduce 任务中能够正确读取。一个可能的方法是使用 Go 的 `encoding/json` 包。以下是将键值对以 JSON 格式写入打开文件的代码：

```
go复制编辑enc := json.NewEncoder(file)
for _, kv := ... {
  err := enc.Encode(&kv)
}
```

并且读取这些文件的方法是：

```
go复制编辑dec := json.NewDecoder(file)
for {
  var kv KeyValue
  if err := dec.Decode(&kv); err != nil {
    break
  }
  kva = append(kva, kv)
}
```

worker 中的 Map 部分可以使用 `ihash(key)` 函数来决定某个给定的键应该分配到哪个 Reduce 任务。

你可以从 `mrsequential.go` 中偷取一些代码，来读取 Map 输入文件、排序 Map 和 Reduce 之间的中间键值对，并将 Reduce 输出存储到文件中。

作为一个 RPC 服务器，协调器会是并发的；不要忘记锁住共享数据。

使用 Go 的竞态检测工具 `-race`，例如使用以下命令运行：

```
bash


复制编辑
go run -race
```

`test-mr.sh` 中有一段注释，告诉你如何**使用 `-race` 运行它**。在我们评分时，我们不会使用竞态检测器。然而，如果你的代码**有竞态问题**，很可能在我们测试时**即使不使用竞态检测器也会失败**。

工作节点**有时需要等待**，例如，Reduce 任务不能在 Map 任务完成之前开始。一个可能的做法是让工作节点**周期性地向协调器请求任务**，在每次请求之间使用 `time.Sleep()` 来休眠。另一种可能的做法是，让**协调器中的相关 RPC 处理程序通过一个循环来等待**，使用 `time.Sleep()` 或 `sync.Cond` 来等待。Go 会为每个 RPC 处理程序启动一个单独的线程，因此一个处理程序的等待不会阻止协调器处理其他 RPC。

协调器无法可靠地区分崩溃的工作节点、尚未完成任务的工作节点和执行太慢的工作节点。你可以做的最好的事情是让协调器等待一段时间，然后重新分配任务给另一个工作节点。在这个实验中，协调器应该等待 10 秒钟；如果超时，协调器可以假设工作节点崩溃（当然，这也可能不对）。

如果你选择实现备份任务（见第 3.6 节），请注意我们会检查你的代码，确保当工作节点执行任务时没有额外调度任务。备份任务应该只在相对较长的时间（例如 10 秒）后调度。

为了测试崩溃恢复，你可以使用 `mrapps/crash.go` 应用程序插件，它会在 Map 和 Reduce 函数中随机退出。

为了确保在崩溃的情况下没有人看到部分写入的文件，MapReduce 论文提到了一种技巧，即使用临时文件并在完全写入后原子地重命名它。你可以使用 `ioutil.TempFile`（或者如果你使用 Go 1.17 或更高版本，可以使用 `os.CreateTemp`）来创建临时文件，并使用 `os.Rename` 来原子地重命名它。

`test-mr.sh` 会在子目录 `mr-tmp` 中运行所有进程，因此，如果出现问题，你想查看中间文件或输出文件，可以去那里查看。你可以临时修改 `test-mr.sh`，让它在失败的测试后退出，以便脚本不会继续测试（并覆盖输出文件）。

`test-mr-many.sh` 会多次运行 `test-mr.sh`，你可能想运行这个脚本来捕捉低概率的错误。你不应该并行运行多个 `test-mr.sh` 实例，因为协调器会重用相同的套接字，导致冲突。

Go 的 RPC 只会发送那些字段名以大写字母开头的结构体字段。子结构体的字段名也必须以大写字母开头。

在调用 RPC 的 `call()` 函数时，回复结构体应包含所有默认值。RPC 调用应该是这样的：

```
go复制编辑reply := SomeType{}
call(..., &reply)
```

不要在调用之前设置 `reply` 中的任何字段。如果你传递了包含非默认字段的回复结构体，RPC 系统可能会悄悄地返回不正确的值。





框架中的两个角色：

1.Woker:

接受两个func  : map 和 reduce 

通过 想cooradinator发送请求，获得需要处理的工作，根据工作类型调用map或者reduce函数处理。

2.coordinator:

维护整个job的推进，包括分配任务给发送请求的worker，记录未处理的文件和已处理文件，将任务分配处理好。最终分发end任务给worker让他们终止。

容错：

记录每个任务执行的时间，如果任务执行超时，让该节点停止任务执行，重新分配节点。





**实现 Map 和 Reduce**：

- Map: 从输入文件读取数据，并产生一系列键值对。
- Reduce: 根据 Map 结果，对相同的键进行聚合。

**并行化执行**：

- 多个 Map 任务和 Reduce 任务可以并行执行，工作者之间需要协调任务的分配。

**容错机制**：

- 如果工作者进程失败，系统应该能自动恢复，重新分配任务。
- 要设计一个机制来处理任务执行失败和工作者崩溃的情况。

**RPC 通信**：

- 通过 RPC（远程过程调用）来实现工作者和协调器之间的通信。
- 协调器发送任务给工作者，工作者执行任务后返回结果。

**文件系统管理**：

- Map 任务的结果会存储到中间文件中，Reduce 任务会根据这些中间结果进行处理。
- 最终结果需要保存在 `mr-out-X` 文件中



coordinator的工作：

1.管理任务状态

2.处理worker的RPC请求

3.检测任务是否超时



worker的工作：

1.向coordinator请求任务  // 以RPC请求的方式请求

2.执行map或者reduce 任务

3.处理文件的输入和输出

4.汇报任务状态//  以RPC请求的方式汇报







# 一.定义协调器的属性

包括：

1.map任务列表

2.reduce任务列表

3.nreduce的值，也就是reduce任务需要分出的桶的值

4.两个阶段的完成情况bool值 allfinishied和mapfinishied
5.同步机制，一个锁（因为RPC调用使用的是goroutine的多线程机制，因此要避免竞态，比如任务分配的冲突）



# 二.定义任务属性

包括：

1.任务的ID，用于当worker执行完任务之后反馈结果的时候方便修改任务完成情况

2.任务状态：分为三个情况：未完成，执行中，和已完成。由于涉及到了容错机制，当一个节点超时时，我们需要将任务重新分配，因此我们需要维护好一个执行中的状态，来观察它是否超时

3.超时时间，用来判断是否超时

4.文件名：这个用于给map任务使用，因为map任务的文件名是人为指定的。



# 三.初始化

mrcoodinator.go中main调用了makecoordinator创建了一个master 并且使之工作

传入的属性有：map阶段的所有文件名，以及nreduce的值

makecoordinator首先创建了一个master实例  

我们需要对其初始化：

1.初始化maptask[]

2.初始化nreduce

3.初始化reducetask

这里的reducetask的数量就是nreduce的数量，也就代表着我们的map将数据拆分成了多少个可以分开进行reduce 的类，这些reduce文件中的数据或许不是完全一样，但是它们的哈希值是一样的。



# 四.worker

有了前面的编写，我们创建了一个协调器，它不断地调用着Done函数期待所有工作被做完的时候退出。

我们需要定义worker的工作，让它能够通过RPC请求调用协调器的函数。

我们需要worker做三件事：

1.请求任务

2.执行任务

3.汇报完成



让worker获得自己的uid作为workerid，这一步暂时觉得是**多余**的。



# 五.请求任务

首先我们需要worker中有一个无限循环，不断向master请求任务。

这需要一个函数，getTask,它返回一个Task
**getTask**内部必然是调用master的RPC GetTask函数。

GetTask函数由于是一个rpc调用，所以接受的参数：args以及reply，返回的是err。

所以我们需要完成args和reply的结构



这里的请求是gettask，所以我们将这两个结构定义为：

GetTaskArgs和GetTaskReply

**GetTaskArgs**应该包含的参数;

1.workerID

**GetTaskReply**应该包含的参数

1.一个filename表示应该处理的文件名

2.tasktype文件类型：其中包含四种：1.map,2.reduce,3.wait,4.exit

3.taskID

4.nreduce

以上四个就可以执行map，**ps:nreduce用来将map的结果分bucket的**

由于reduce涉及到访问多个文件的shuffle，每个文件的名字格式：mr-X-Y

其中X为map编号，Y为reduce编号

要想执行reduce，我们不光需要拿到taskID，还需要知道map任务的总数，也就是X的最大值。

这样我们就可以遍历所有的mr-i - taskID的文件，拿到对应的数据

因为我们需要如下结构：

5.maptasknum  这个maptasknum就是map任务的总数，也就是len(c.MapTasks)



# 6.编写GetTask函数

GetTask函数接受args和reply。

0.加锁，由于RPC是使用的goroutine来响应，因此需要避免竞态。

1.检查超时任务

将超时任务设置为未完成状态，同时检查当前任务状态，更新两阶段任务完成情况

2.检查是否已经完成mapreduce，分发Exit任务。



3.检查第一阶段任务，如果有未执行，那么分发  return

4.如果第一阶段任务已经完成，检查第二阶段任务，如果有未执行，那么分发 return 

5.如果出现了不满足上述两种情况的情况 分发wait任务



**分发maptask:**

找到协调器的maptasks中的Status为空闲的一个任务。

maptask需要的参数：

1.filename
2.taskID（用于执行完毕之后的提交阶段的报告）

3.nreduce（用于分配bucket）

4.tasktype

之后我们需要将协调器维护的maptask数组中对应位置的任务的Status改为Progressing并且记录下执行任务的时间。

如果找不到空闲任务，那么我们就设置一个wait任务

分发reducetask:

如果map已经结束，那么我们找到reducetasks中的一个空闲任务

需要的参数：

taskID

maptasknum (用于遍历所有map产生的对应taskID的文件)

tasktype

同样;之后我们需要将协调器维护的maptask数组中对应位置的任务的Status改为Progressing并且记录下执行任务的时间。

如果找不到空闲任务，那么我们就设置一个wait任务

最后如果跳过了这两个条件进入了程序末尾，那么我们就fatal。



# 7.timeout检查

1.定义一个过期时间，mapreduce 原文中为10s.

2.同样先查看map任务是否执行完， 如果未执行完，我们就检查Map任务

如果执行完，我们顺手更新map任务状态。

3.检查reduce 任务是否执行完。

4.将超时任务的状态设置为Idle



# 8.worker中执行请求到的任务。

使用switch 处理四种不同的任务请求
1.map : 调用domap函数，最后记得report

2.reduce ：调用doreduce 函数，最后记得report

3.wait: sleep 0.5s

4.exit: return



# 9.Report函数：

我们这里设定只有当完成了工作之后才能report，

report函数需要接受的参数：workerID,tasktype,TaskID.

report调用RPC调用协调器的ReportTask函数。

需要设定两个参数Args和Reply

ReportTaskArgs:

workerID

taskType

TaskId

ReportTaskReply:

 ok bool  表示执行完成

不太需要参数



# 10.domap函数：

domap函数是worker.go的本地程序，它接受gettask的响应参数，也就是taskreply，并且接受用户自定义mapf，以及workerID，之后调用mapf处理task。

GetTaskReply的参数：

```
FileName   string
TaskID     int
TaskType   string
NReduce    int
MapTaskNum int
```

其中map函数需要使用的：

Filename  TaskID NReduce



执行过程：
1.通过filename打开file

Read其中的文件得到content

使用mapf进行map: **mapf接受filename和string类型的content，返回一个kv列表**

PS：kv是一个自定义的结构体，包含了k和v

我们要做的：

1.创建nReduce 个bucket，这里表现为二维数组。

之后我们使用mapf得到一个kv列表，

for 每个kv

​	我们对K进行哈希，将它映射到对应的bucket中，将它放入对应的bucket

之后我们将这若干个bucket存储起来。格式为：mr-X-Y（这里，我们使用的方法：首先创建一个临时文件，之后将内容写入再改名，这样就可以避免突然的崩溃导致带着完整名字的文件内容不全）



# 11.doReduce函数

接受gettask响应的参数，以及reducef和workerID

能够使用的参数：

TaskID,MapTaskNum

**shuffle**

1.读取所有的mr-i-TaskID的文件

将文件内容以json格式解码到一个intermediate kv数组里面

2.将intermediate的kv数组按照key排序

3.

for 整个 Intermediate

将相同的key的值放到同一个数组中



对这个数据进行reduce，得到输出为一个output string

4.创建一个临时文件，将所有output 写入其中。

将output 改名为mr-out -X。

5.report任务完成。

