import pandas as pd
import numpy as np
import torch
import torch.nn as nn
from torch.utils.data import Dataset, DataLoader
from sklearn.preprocessing import MinMaxScaler, OneHotEncoder
import matplotlib.pyplot as plt

# 参数配置
SEQ_LENGTH = 30
BATCH_SIZE = 64
HIDDEN_SIZE = 64
NUM_LAYERS = 1
EPOCHS = 200
LEARNING_RATE = 1e-4

def convert_time(raw_time):
    """时间格式转换核心函数"""
    str_time = str(int(raw_time)).zfill(12)  # 处理科学计数法
    return pd.to_datetime(str_time, format='%Y%m%d%H%M', errors='coerce')

def load_data(file_path):
    """数据加载与特征工程"""
    data = pd.read_csv(file_path)
    
    # 转换时间列
    data['current_time'] = data['current_time'].apply(convert_time)
    data = data.dropna(subset=['current_time']).reset_index(drop=True)
    
    # 时间特征提取
    data['hour'] = data['current_time'].dt.hour
    data['minute'] = data['current_time'].dt.minute
    data['dayofweek'] = data['current_time'].dt.dayofweek
    
    # 周期编码
    data['hour_sin'] = np.sin(2 * np.pi * data['hour'] / 24)
    data['hour_cos'] = np.cos(2 * np.pi * data['hour'] / 24)
    data['minute_sin'] = np.sin(2 * np.pi * data['minute'] / 60)
    data['minute_cos'] = np.cos(2 * np.pi * data['minute'] / 60)
    
    # One-hot编码星期（修复点）
    onehot = OneHotEncoder(sparse_output=False)
    week_onehot = onehot.fit_transform(data[['dayofweek']])
    week_cols = [f'week_{i}' for i in range(7)]  # 明确定义列名
    data[week_cols] = week_onehot
    
    return data, week_cols  # 返回列名信息

def preprocess_data(data, week_cols):  # 接收列名参数
    """数据预处理与特征选择"""
    split_idx = int(0.8 * len(data))
    train = data.iloc[:split_idx]
    test = data.iloc[split_idx:]
    
    # 归一化
    scaler = MinMaxScaler()
    train['traffic'] = scaler.fit_transform(train[['traffic']])
    test['traffic'] = scaler.transform(test[['traffic']])
    
    # 特征选择（使用传入的week_cols）
    features = ['hour_sin', 'hour_cos', 'minute_sin', 'minute_cos'] + week_cols + ['traffic']
    return train[features], test[features], scaler

def create_sequences(data, seq_length):
    """创建时间序列数据集"""
    xs, ys = [], []
    for i in range(len(data)-seq_length):
        x = data[i:i+seq_length]
        y = data[i+seq_length, -1]  # 预测下一个时间步的流量
        xs.append(x)
        ys.append(y)
    return np.array(xs), np.array(ys)

class TrafficDataset(Dataset):
    """PyTorch数据集类"""
    def __init__(self, features, targets):
        self.features = torch.FloatTensor(features)
        self.targets = torch.FloatTensor(targets).view(-1, 1)
    
    def __len__(self):
        return len(self.features)
    
    def __getitem__(self, idx):
        return self.features[idx], self.targets[idx]

class LSTMModel(nn.Module):
    """LSTM模型定义"""
    def __init__(self, input_size, hidden_size, num_layers):
        super().__init__()
        self.lstm = nn.LSTM(input_size, hidden_size, num_layers, batch_first=True)
        self.fc = nn.Linear(hidden_size, 1)
    
    def forward(self, x):
        out, _ = self.lstm(x)
        return self.fc(out[:, -1, :])

if __name__ == "__main__":
    # 数据加载
    raw_data, week_columns = load_data("/Users/bencyq/Desktop/code/cpn_controller/pkg/python/lstm.csv")  # 获取列名
    
    # 数据预处理
    train_data, test_data, scaler = preprocess_data(raw_data, week_columns)  # 传递列名
    
    # 序列创建
    X_train, y_train = create_sequences(train_data.values, SEQ_LENGTH)
    X_test, y_test = create_sequences(test_data.values, SEQ_LENGTH)
    
    # 数据加载器
    train_dataset = TrafficDataset(X_train, y_train)
    test_dataset = TrafficDataset(X_test, y_test)
    train_loader = DataLoader(train_dataset, batch_size=BATCH_SIZE, shuffle=True)
    test_loader = DataLoader(test_dataset, batch_size=BATCH_SIZE, shuffle=False)
    
    # 模型初始化
    model = LSTMModel(
        input_size=X_train.shape[-1],
        hidden_size=HIDDEN_SIZE,
        num_layers=NUM_LAYERS
    )
    criterion = nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=LEARNING_RATE)
    
    # 训练循环
    for epoch in range(EPOCHS):
        model.train()
        total_loss = 0
        for X_batch, y_batch in train_loader:
            optimizer.zero_grad()
            outputs = model(X_batch)
            loss = criterion(outputs, y_batch)
            loss.backward()
            optimizer.step()
            total_loss += loss.item()
        print(f"Epoch {epoch+1}/{EPOCHS}, Loss: {total_loss/len(train_loader):.4f}")
    
    # 测试评估
    model.eval()
    test_preds, test_true = [], []
    with torch.no_grad():
        for X_batch, y_batch in test_loader:
            outputs = model(X_batch)
            test_preds.extend(outputs.numpy())
            test_true.extend(y_batch.numpy())
    
    # 反归一化
    test_preds = scaler.inverse_transform(np.array(test_preds))
    test_true = scaler.inverse_transform(np.array(test_true))
    
    # 评估指标
    rmse = np.sqrt(np.mean((test_preds - test_true)**2))
    mae = np.mean(np.abs(test_preds - test_true))
    print(f"Test RMSE: {rmse:.2f} b/s, MAE: {mae:.2f} b/s")
    
    # 可视化
    plt.figure(figsize=(12, 6))
    plt.plot(test_true, label="Actual Traffic")
    plt.plot(test_preds, label="Predicted Traffic")
    plt.title("Traffic Prediction Comparison")
    plt.xlabel("Time Steps")
    plt.ylabel("Traffic (b/s)")
    plt.legend()
    plt.show()