import time
import pickle
import numpy as np

# 加载模型（可选，用于验证）
with open('random_forest_weight.pt', 'rb') as file:
    loaded_model = pickle.load(file)
start_time=time.time()
prediction = loaded_model.predict(np.array([[258547251200.00,0,0,0.022269968,0.065642652,22.76492574]]))
print(f"加载的模型预测结果: {prediction}")
print((time.time()-start_time)/10)
