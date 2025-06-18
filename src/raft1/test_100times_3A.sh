#!/bin/bash

# 获取当前日期时间
CURRENT_TIME=$(date "+%Y%m%d_%H%M%S")
#test_results_100_${CURRENT_TIME}.txt
# 定义输出文件名，将当前时间加到文件名中
OUTPUT_FILE="testResult/test_results_100_${CURRENT_TIME}.txt"

# 执行测试100次
for i in {1..100}
do
  echo "============================== Test Run $i ==============================" >> $OUTPUT_FILE
  go test -run 3A >> $OUTPUT_FILE
  echo "" >> $OUTPUT_FILE  # 每次执行后添加一个空行
done

echo "Test executions completed. Results are saved in $OUTPUT_FILE."