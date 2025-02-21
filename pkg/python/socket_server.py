import json
import socket
import os
import sys
sys.path.append(os.path.dirname(os.path.dirname(__file__)))
import random_forest_predict

def get_project_root():
    # 获取当前工作目录
    current_dir = os.getcwd()

    while True:
        # 拼接当前目录和 go.mod 文件的路径
        go_mod_path = os.path.join(current_dir, 'go.mod')
        # 检查 go.mod 文件是否存在
        if os.path.exists(go_mod_path):
            return current_dir
        # 获取当前目录的父目录
        parent_dir = os.path.dirname(current_dir)
        # 如果父目录和当前目录相同，说明已经到了根目录
        if parent_dir == current_dir:
            raise FileNotFoundError("go.mod not found")
        # 更新当前目录为父目录
        current_dir = parent_dir


def init_socket():
    # 创建并绑定到 UNIX 域套接字
    socket_path = get_project_root()+"/pkg/python/rfp.sock"
    if os.path.exists(socket_path):
        os.remove(socket_path)
    
    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(socket_path)
    server.listen(1)
    print("INFO: socket server started, waiting for connection...")

    try:
        while True:  # 添加主监听循环
            conn = None
            try:
                # 接受新连接
                conn, addr = server.accept()
                # print(f"DEBUG: New connection established")

                # 接收数据
                data = conn.recv(65536)
                if data:
                    # print(f"INFO: Received from Go: {data.decode()}")
                    # 解析json
                    data=json.loads(data.decode())
                    data[1].extend(['none' for _ in range(3-len(data[1]))])  # 补全data[1]可能存在的空缺，即模型的输入个数可能为1,2,3
                    data[0]=[float(ele) for ele in data[0]]
                    # 调用预测器
                    response:str=random_forest_predict.predict(data[1],data[0])
                    conn.sendall(response.encode())
                    # print(f"DEBUG: Sent to Go: {response}")
                
            except ConnectionResetError:
                print("WARNING: Client disconnected unexpectedly")
            except Exception as e:
                print(f"ERROR: {str(e)}")
            finally:
                if conn:
                    conn.close()
                # print("DEBUG: Waiting for new connection...")

    except KeyboardInterrupt:
        print("\nINFO: Shutting down server...")
    finally:
        server.close()
        if os.path.exists(socket_path):
            os.remove(socket_path)
        print("INFO: Server shutdown complete")

if __name__ == "__main__":
    init_socket()
