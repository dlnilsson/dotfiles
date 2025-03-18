import subprocess
import time
from rofi import Rofi
import pyperclip
import re

CACHE_DURATION = 10  # seconds
cache = {
    "timestamp": 0,
    "data": None,
}

def get_ykman_accounts():
    global cache
    current_time = time.time()

    if cache["data"] is not None and (current_time - cache["timestamp"] < CACHE_DURATION):
        return cache["data"]

    try:
        result = subprocess.run(['ykman', 'oath', 'accounts', 'code'],
                                stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

        if result.returncode != 0:
            raise RuntimeError(f"Error running ykman: {result.stderr.strip()}")

        accounts = result.stdout.strip().split('\n')

        cache["data"] = accounts
        cache["timestamp"] = current_time

        return accounts
    except FileNotFoundError:
        raise RuntimeError("ykman not found. Make sure it is installed and in your PATH.")

def extract_code(account_line):
    match = re.search(r'\b(\d{6})\b$', account_line)
    if match:
        return match.group(1)
    return None

def main():
    r = Rofi()
    try:
        accounts = get_ykman_accounts()
    except RuntimeError as e:
        subprocess.run(['notify-send', str(e)])
        return

    index, _ = r.select('Choose an account', accounts, rofi_args=['-i'])

    if index != -1:
        selected_account = accounts[index]
        code = extract_code(selected_account)
        print(f"select_account {selected_account}")
        if code:
            pyperclip.copy(code)
            name = selected_account.replace(code, '').strip()
            subprocess.run(['notify-send', f"Token for {name} copied to clipboard"])
        else:
            subprocess.run(['notify-send', "Failed to extract code from the selected account."])
    else:
        subprocess.run(['notify-send', "No selection made"])

if __name__ == "__main__":
    main()
