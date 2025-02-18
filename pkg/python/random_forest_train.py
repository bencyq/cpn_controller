import pandas as pd
from sklearn.ensemble import RandomForestRegressor
from sklearn.metrics import r2_score
import time
import pickle

# 1. 读取训练和测试数据
train_data = pd.read_csv('train.csv') 
test_data = pd.read_csv('test.csv')  

# 2. 检查并处理缺失值
if train_data.isnull().sum().any() or test_data.isnull().sum().any():
    print("存在缺失值，开始处理...")
    train_data = train_data.dropna()
    test_data = test_data.dropna()
    print("缺失值已处理，删除包含缺失值的行。")

X_train = train_data[['Flops1', 'Flops2', 'Flops3', 'performance1', 'performance2', 'performance3']]
y_train = train_data[['averagetime1', 'averagetime2', 'averagetime3']]

X_test = test_data[['Flops1', 'Flops2', 'Flops3', 'performance1', 'performance2', 'performance3']]
y_test = test_data[['averagetime1', 'averagetime2', 'averagetime3']]

model = RandomForestRegressor(n_estimators=100, random_state=42)
model.fit(X_train, y_train)
with open('random_forest_weight.pt', 'wb') as file:
    pickle.dump(model, file)
start_time=time.time()
y_pred = model.predict(X_test)
comparison = pd.DataFrame({
    'True Time1': y_test['averagetime1'],
    'Predicted Time1': y_pred[:, 0],
    'True Time2': y_test['averagetime2'],
    'Predicted Time2': y_pred[:, 1],
    'True Time3': y_test['averagetime3'],
    'Predicted Time3': y_pred[:, 2]
})
print(comparison)
r2_overall = r2_score(y_test.values.flatten(), y_pred.flatten())
print(f"Overall R²: {r2_overall}")
print((time.time()-start_time)/10)