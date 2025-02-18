import socket
import os

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


def main():
    # 创建并绑定到 UNIX 域套接字
    socket_path = get_project_root()+"/pkg/python/rfp.sock"
    if os.path.exists(socket_path):
        os.remove(socket_path)
    
    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(socket_path)
    server.listen(1)
    print("INFO: socket server started, waiting for connection...")

    # 接受连接
    conn, addr = server.accept()
    print("INFO: Connection established!")

    # 接收来自 Go 进程的数据
    data = conn.recv(65536)
    print("INFO: Received from Go:", data.decode())

    # 发送响应数据给 Go 进程
    response = "Hello from Python"
    conn.sendall(response.encode())
    print("INFO: Sent to Go:", response)

    # 关闭连接
    conn.close()

if __name__ == "__main__":
    main()
