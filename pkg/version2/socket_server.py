import socket
import os
import json

def main():
    # 创建并绑定到 UNIX 域套接字
    socket_path = "pkg/version2/scheduler.sock"
    if os.path.exists(socket_path):
        os.remove(socket_path)
    
    server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    server.bind(socket_path)
    server.listen(1)
    print("INFO: Waiting for connection...")

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
