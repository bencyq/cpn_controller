from copy import deepcopy
import copy
import os
import random
import sys
import numpy as np
import pandas as pd
from sklearn.ensemble import RandomForestRegressor
from sklearn.metrics import r2_score, mean_absolute_error  # 添加MAE导入
import time
import pickle
import itertools

from sympy import O
from tables import test
sys.path.append(os.path.dirname(os.path.dirname(__file__)))
from utils import get_project_root

# 读取训练和测试数据
root = get_project_root()
train_data = pd.read_csv(root+'/pkg/python/'+'train.csv') 
test_data = pd.read_csv(root+'/pkg/python/'+'test.csv')  

# 检查并处理缺失值
if train_data.isnull().sum().any() or test_data.isnull().sum().any():
    print("存在缺失值，开始处理...")
    train_data = train_data.dropna()
    test_data = test_data.dropna()
    print("缺失值已处理，删除包含缺失值的行。")

train_data=train_data.values
test_data=test_data.values

def my_shuffle(data:list):
    original_list = [0, 1, 2]
    permutations = list(itertools.permutations(original_list))
    new_data=np.empty((0,9))
    for ele in data:
        for p in permutations:
            tmp=copy.deepcopy(ele)
            tmp[0],tmp[6]=ele[p[0]],ele[p[0]+6]
            tmp[1],tmp[7]=ele[p[1]],ele[p[1]+6]
            tmp[2],tmp[8]=ele[p[2]],ele[p[2]+6]
            tmp = tmp.reshape(1, -1)
            new_data=np.concatenate((new_data,tmp),axis=0)
    return np.concatenate((data, new_data), axis=0)

train_data=my_shuffle(train_data)
test_data=my_shuffle(test_data)

model = RandomForestRegressor(n_estimators=100, random_state=42)
model.fit(train_data[:,0:6], train_data[:,6:])
with open(root+'/pkg/python/'+'random_forest_weight.pt', 'wb') as file:
    pickle.dump(model, file)

# 测试
def compare(data):
    start_time=time.time()
    y_pred = model.predict(data[:,0:6])
    comparison = pd.DataFrame({
        'True Time1': data[:,6],
        'Predicted Time1': y_pred[:, 0],
        'True Time2': data[:,7],
        'Predicted Time2': y_pred[:, 1],
        'True Time3': data[:,8],
        'Predicted Time3': y_pred[:, 2]
    })
    print(comparison)
    r2_overall = r2_score(data[:,6:], y_pred)
    print(f"Overall R²: {r2_overall}")
    
    # 新增MAE计算逻辑
    mae_time1 = mean_absolute_error(data[:,6], y_pred[:,0])
    mae_time2 = mean_absolute_error(data[:,7], y_pred[:,1])
    mae_time3 = mean_absolute_error(data[:,8], y_pred[:,2])
    mae_overall = mean_absolute_error(data[:,6:].flatten(), y_pred.flatten())
    
    print(f"MAE Time1: {mae_time1:.4f}")
    print(f"MAE Time2: {mae_time2:.4f}")
    print(f"MAE Time3: {mae_time3:.4f}")
    print(f"Overall MAE: {mae_overall:.4f}")
    
    print(f"Execution time: {(time.time()-start_time)/10:.4f}s")

compare(train_data)
compare(test_data)
