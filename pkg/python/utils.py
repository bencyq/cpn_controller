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
