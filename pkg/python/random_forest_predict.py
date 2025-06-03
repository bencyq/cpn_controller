import time
import pickle
import numpy as np
from utils import get_project_root

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

root = get_project_root()

# 加载模型（可选，用于验证）
with open(root+'/pkg/python/random_forest_weight.pt', 'rb') as file:
    loaded_model = pickle.load(file)
    print("INFO: random forest model loaded")

def predict(models,benchmark):
    prediction :np.ndarray= loaded_model.predict(np.array([[model_FLOPs[models[0]],model_FLOPs[models[1]],model_FLOPs[models[2]],benchmark[0],benchmark[1],benchmark[2]]]))
    return ','.join(map(str,prediction.flatten()))
    # return ",".join([f"{x:.4f}" for x in prediction[0]])

if __name__=="__main__":
    # print(predict(['densenet121','none','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['none','densenet121','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['none','none','densenet121'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet169','none','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['none','densenet169','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['none','none','densenet169'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet169','densenet201','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet169','densenet201','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    # print(predict(['densenet169','densenet201','none'],[0.014736412,0.051585681,36.71008748]))  # A6000的硬件信息
    # print(predict(['densenet201','densenet169','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet201','densenet169','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    # print(predict(['densenet201','densenet169','none'],[0.014736412,0.051585681,36.71008748]))  # A6000的硬件信息
    # print(predict(['densenet201','densenet169','none'],[0.022269968,0.065642652,42.71008748]))  # P100的硬件信息
    # print(predict(['densenet201','densenet169','none'],[0.022269968,0.065642652,0]))  # P100的硬件信息(llama3的为0)
    # print(predict(['densenet201','densenet121','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet121','densenet201','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    # print(predict(['densenet201','none','none'],[0.017401139,0.054753114,18.97077145]))  # A100的硬件信息
    print(predict(['resnet152','none','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    print(predict(['resnet152','glm4','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    print(predict(['none','glm4','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    print(predict(['resnet152','none','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    print(predict(['resnet152','qwen2.5','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息
    print(predict(['none','qwen2.5','none'],[0.01656568,0.065646874,21.20620537]))  # 4090的硬件信息