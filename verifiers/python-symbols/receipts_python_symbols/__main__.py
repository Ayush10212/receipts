"""Entry point: python -m receipts_python_symbols"""

import sys
from receipts_python_symbols.rpc import run_loop

if __name__ == "__main__":
    run_loop(sys.stdin, sys.stdout)
