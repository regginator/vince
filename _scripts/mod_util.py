import os
import sys
import subprocess

def run(args=[], env={}):
    print(f"+ {' '.join(args)}")

    cmd_env=os.environ.copy()
    for key, val in env.items():
        cmd_env[key] = val

    try:
        subprocess.run(args, check=True, env=cmd_env)
    except subprocess.CalledProcessError as ret:
        print(f"\n^ command above failed with exit code {ret.returncode}, abort..")
        sys.exit(ret.returncode)

def cd_root():
    os.chdir(os.path.dirname(os.path.abspath(__file__)) + "/../")

def getenv(variable):
    ret = os.environ.get(variable)
    if ret is None:
        ret = ""

    return ret
