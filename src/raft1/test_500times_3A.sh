#!/bin/bash

# 定义输出文件
OUTPUT_FILE="test_results_500.txt"

# 执行测试500次
for i in {1..500}
do
  echo "============================== Test Run $i ==============================" >> $OUTPUT_FILE
  go test -run 3A >> $OUTPUT_FILE
  echo "" >> $OUTPUT_FILE  # 每次执行后添加一个空行
done

echo "Test executions completed. Results are saved in $OUTPUT_FILE."