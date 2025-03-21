from copy import deepcopy
import copy
import os
import random
import sys
import numpy as np
import pandas as pd
from sklearn.multioutput import MultiOutputRegressor  # 新增多输出包装器
from xgboost import XGBRegressor  # 替换模型导入
from sklearn.metrics import r2_score
import time
import pickle
import itertools

sys.path.append(os.path.dirname(os.path.dirname(__file__)))
from utils import get_project_root

# 读取数据部分保持不变
root = get_project_root()
train_data = pd.read_csv(root+'/pkg/python/'+'train.csv') 
test_data = pd.read_csv(root+'/pkg/python/'+'test.csv')  

# 缺失值处理保持不变
if train_data.isnull().sum().any() or test_data.isnull().sum().any():
    print("存在缺失值，开始处理...")
    train_data = train_data.dropna()
    test_data = test_data.dropna()
    print("缺失值已处理，删除包含缺失值的行。")

train_data = train_data.values
test_data = test_data.values

# 数据增强函数保持不变
def my_shuffle(data:list):
    original_list = [0, 1, 2]
    permutations = list(itertools.permutations(original_list))
    new_data = np.empty((0,9))
    for ele in data:
        for p in permutations:
            tmp = copy.deepcopy(ele)
            tmp[0], tmp[6] = ele[p[0]], ele[p[0]+6]
            tmp[1], tmp[7] = ele[p[1]], ele[p[1]+6]
            tmp[2], tmp[8] = ele[p[2]], ele[p[2]+6]
            tmp = tmp.reshape(1, -1)
            new_data = np.concatenate((new_data,tmp), axis=0)
    return np.concatenate((data, new_data), axis=0)

train_data = my_shuffle(train_data)
test_data = my_shuffle(test_data)

# 模型定义部分修改为XGBoost
model = MultiOutputRegressor(
    XGBRegressor(
        n_estimators=100,
        learning_rate=0.1,  # 新增学习率参数
        max_depth=3,        # 调整树深
        subsample=0.8,      # 新增行采样
        colsample_bytree=0.8, # 新增列采样
        random_state=42
    )
)

# 训练过程保持不变
model.fit(train_data[:, 0:6], train_data[:, 6:])

# 模型保存格式保持不变
with open(root+'/pkg/python/'+'xgboost_weight.pt', 'wb') as file:
    pickle.dump(model, file)

# 测试函数保持不变
def compare(data):
    start_time = time.time()
    y_pred = model.predict(data[:, 0:6])
    comparison = pd.DataFrame({
        'True Time1': data[:, 6],
        'Predicted Time1': y_pred[:, 0],
        'True Time2': data[:, 7],
        'Predicted Time2': y_pred[:, 1],
        'True Time3': data[:, 8],
        'Predicted Time3': y_pred[:, 2]
    })
    print(comparison)
    r2_overall = r2_score(data[:, 6:], y_pred)
    print(f"Overall R²: {r2_overall}")
    print((time.time() - start_time)/10)

compare(train_data)
compare(test_data)