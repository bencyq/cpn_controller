import pandas as pd
import numpy as np
import torch
import torch.nn as nn
from torch.utils.data import Dataset, DataLoader
from sklearn.preprocessing import MinMaxScaler, OneHotEncoder
import matplotlib.pyplot as plt

# 参数配置
SEQ_LENGTH = 60  # 历史时间窗口长度
BATCH_SIZE = 64
HIDDEN_SIZE = 128
NUM_LAYERS = 2
EPOCHS = 50
LEARNING_RATE = 0.001

# 时间转换函数（新增核心部分）
def convert_time(raw_time):
    """将数字时间202503100455转换为datetime对象"""
    str_time = str(int(raw_time)).zfill(12)  # 处理科学计数法/字符串类型
    return pd.to_datetime(str_time, format='%Y%m%d%H%M', errors='coerce')

# 数据加载与预处理（修改时间处理逻辑）
def load_data(file_path):
    # 读取CSV数据
    data = pd.read_csv(file_path)
    
    # 核心修改：转换时间列
    data['current_time'] = data['current_time'].apply(convert_time)
    
    # 删除无效时间记录
    data = data.dropna(subset=['current_time']).reset_index(drop=True)
    
    # 提取时间特征
    data['hour'] = data['current_time'].dt.hour
    data['minute'] = data['current_time'].dt.minute
    data['dayofweek'] = data['current_time'].dt.dayofweek
    
    # 周期编码
    data['hour_sin'] = np.sin(2 * np.pi * data['hour'] / 24)
    data['hour_cos'] = np.cos(2 * np.pi * data['hour'] / 24)
    data['minute_sin'] = np.sin(2 * np.pi * data['minute'] / 60)
    data['minute_cos'] = np.cos(2 * np.pi * data['minute'] / 60)
    
    # One-hot编码星期
    onehot = OneHotEncoder(sparse_output=False)
    week_onehot = onehot.fit_transform(data[['dayofweek']])
    week_cols = [f'week_{i}' for i in range(7)]
    data[week_cols] = week_onehot
    
    return data

# 数据集划分与归一化（保持不变）
def preprocess_data(data):
    split_idx = int(0.8 * len(data))
    train = data.iloc[:split_idx]
    test = data.iloc[split_idx:]
    
    scaler = MinMaxScaler()
    train['traffic'] = scaler.fit_transform(train[['traffic']])
    test['traffic'] = scaler.transform(test[['traffic']])
    
    features = ['hour_sin', 'hour_cos', 'minute_sin', 'minute_cos'] + week_cols + ['traffic']
    return train[features], test[features], scaler

# 创建序列数据集（保持不变）
def create_sequences(data, seq_length):
    xs, ys = [], []
    for i in range(len(data)-seq_length):
        x = data[i:i+seq_length]
        y = data[i+seq_length, -1]
        xs.append(x)
        ys.append(y)
    return np.array(xs), np.array(ys)

# PyTorch数据集（保持不变）
class TrafficDataset(Dataset):
    def __init__(self, features, targets):
        self.features = torch.FloatTensor(features)
        self.targets = torch.FloatTensor(targets).view(-1, 1)
    
    def __len__(self):
        return len(self.features)
    
    def __getitem__(self, idx):
        return self.features[idx], self.targets[idx]

# LSTM模型（保持不变）
class LSTMModel(nn.Module):
    def __init__(self, input_size, hidden_size, num_layers):
        super().__init__()
        self.lstm = nn.LSTM(input_size, hidden_size, num_layers, batch_first=True)
        self.fc = nn.Linear(hidden_size, 1)
    
    def forward(self, x):
        out, _ = self.lstm(x)
        return self.fc(out[:, -1, :])

# 主程序（新增数据校验）
if __name__ == "__main__":
    # 示例数据校验
    test_time = 202503100455  # 应该转换为2025-03-10 04:55:00
    print("时间转换测试:", convert_time(test_time))
    
    # 加载数据
    data = load_data("traffic_data.csv")
    print("\n数据样例:")
    print(data[['current_time', 'traffic']].head())
    
    # 预处理
    train_data, test_data, scaler = preprocess_data(data)
    
    # 创建序列
    X_train, y_train = create_sequences(train_data.values, SEQ_LENGTH)
    X_test, y_test = create_sequences(test_data.values, SEQ_LENGTH)
    print(f"\n训练集形状: {X_train.shape}, 测试集形状: {X_test.shape}")
    
    # 数据加载器
    train_dataset = TrafficDataset(X_train, y_train)
    test_dataset = TrafficDataset(X_test, y_test)
    train_loader = DataLoader(train_dataset, batch_size=BATCH_SIZE, shuffle=True)
    test_loader = DataLoader(test_dataset, batch_size=BATCH_SIZE, shuffle=False)
    
    # 初始化模型
    model = LSTMModel(input_size=X_train.shape[-1], 
                     hidden_size=HIDDEN_SIZE,
                     num_layers=NUM_LAYERS)
    criterion = nn.MSELoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=LEARNING_RATE)
    
    # 训练循环
    print("\n开始训练...")
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
    
    # 计算指标
    rmse = np.sqrt(np.mean((test_preds - test_true)**2))
    mae = np.mean(np.abs(test_preds - test_true))
    print(f"\n测试结果: RMSE={rmse:.2f} b/s, MAE={mae:.2f} b/s")
    
    # 可视化
    plt.figure(figsize=(12, 6))
    plt.plot(test_true, label="实际流量")
    plt.plot(test_preds, label="预测流量")
    plt.title("流量预测对比 (反归一化后)")
    plt.xlabel("时间步")
    plt.ylabel("流量 (b/s)")
    plt.legend()
    plt.show()