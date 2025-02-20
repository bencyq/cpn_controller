import time
import pickle
import numpy as np
model_FLOPs={
"none": 0,
"llama3":	3842385117184.03,
"qwen2.5":	3619986341888.0,
"vgg11":	7609140224.00,
"vgg16":	15470314496.00,
"vgg19":	19632112640.00,
"resnet18":	1824033792.00,
"resnet50":	4133742592.00,
"resnet101":	7866435584.00,
"resnet152":	11603945472.00,
"densenet121":	2897007104.00,
"densenet169":	3436117120.00,
"densenet201":	4390381952.00,
"glm4":	4494683275264.00,
"yolov8n":	8857548800.00,
"yolov8m":	79320422400.00,
"yolov8x":	258547251200.00}

# 加载模型（可选，用于验证）
with open('/cyq/cpn-controller/pkg/python/random_forest_weight.pt', 'rb') as file:
    loaded_model = pickle.load(file)
    print("INFO: random forest model loaded")

def predict(models,benchmark):
    prediction :np.ndarray= loaded_model.predict(np.array([[model_FLOPs[models[0]],model_FLOPs[models[1]],model_FLOPs[models[2]],benchmark[0],benchmark[1],benchmark[2]]]))
    return ','.join(map(str,prediction.flatten()))
    # return ",".join([f"{x:.4f}" for x in prediction[0]])

if __name__=="__main__":
    print(predict(['densenet169','llama3','yolov8x'],[0.01656568,0.065646874,21.20620537]))
