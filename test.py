import sys
import os
import subprocess
import requests
from bs4 import BeautifulSoup

# ANSI エスケープシーケンス定数
GREEN = "\033[32m"
RED = "\033[31m"
YELLOW = "\033[33m"
BOLD = "\033[1m"
RESET = "\033[0m"

def fetch_samples(url):
    """問題ページから入出力例のペアを抽出する"""
    headers = {
        "User-Agent": "Mozilla/5.0 (compatible; AtCoderTestRunner/1.0)"
    }
    res = requests.get(url, headers=headers)
    if res.status_code != 200:
        print(f"[Error] ページの取得に失敗しました (Status: {res.status_code}, URL: {url})")
        sys.exit(1)

    soup = BeautifulSoup(res.text, "html.parser")
    statement = soup.find("div", id="task-statement") or soup
    
    inputs = {}
    outputs = {}

    for section in statement.find_all("section"):
        h3 = section.find("h3")
        pre = section.find("pre")
        if not h3 or not pre:
            continue

        title = h3.get_text(strip=True)
        if "入力例" in title:
            num = "".join(filter(str.isdigit, title))
            if num:
                inputs[int(num)] = pre.get_text()
        elif "出力例" in title:
            num = "".join(filter(str.isdigit, title))
            if num:
                outputs[int(num)] = pre.get_text()

    samples = []
    for num in sorted(inputs.keys()):
        if num in outputs:
            samples.append((num, inputs[num], outputs[num]))
    return samples

def run_test(target_script, samples):
    """各サンプルケースに対してスクリプトを実行して検証する"""
    if not os.path.exists(target_script):
        print(f"[Error] 実行対象のスクリプト '{target_script}' が見つかりません。")
        sys.exit(1)

    all_passed = True
    for num, in_data, expected_out in samples:
        print(f"--- Sample {num} ---")
        try:
            proc = subprocess.run(
                [sys.executable, target_script],
                input=in_data,
                text=True,
                capture_output=True,
                timeout=5
            )
        except subprocess.TimeoutExpired:
            print(f"[Result] {YELLOW}{BOLD}TLE{RESET} (Time Limit Exceeded)")
            all_passed = False
            continue

        if proc.returncode != 0:
            print(f"[Result] {RED}{BOLD}RE{RESET} (Runtime Error) exit code: {proc.returncode}")
            print(proc.stderr.strip())
            all_passed = False
            continue

        actual_out = proc.stdout.rstrip()
        expected = expected_out.rstrip()

        if actual_out == expected:
            print(f"{GREEN}{BOLD}AC{RESET}")
        else:
            print(f"{RED}{BOLD}WA{RESET}")
            print(f"Expected:\n{expected}")
            print(f"Actual:\n{actual_out}")
            all_passed = False

    print("\n" + ("=" * 20))
    if all_passed:
        print(f"{GREEN}{BOLD}All Samples Passed! (AC){RESET}")
    else:
        print(f"{RED}{BOLD}Some Samples Failed. (WA / RE / TLE){RESET}")


CONTEST_ID = "abc474"

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("invalid args")
        sys.exit(1)
    task_label = sys.argv[1].lower()
    task_id = f"{CONTEST_ID}_{task_label}"
    url = f"https://atcoder.jp/contests/{CONTEST_ID}/tasks/{task_id}"
    if len(sys.argv) >= 3:target_code = sys.argv[2]
    else:
        default_name = f"{task_label}.py"
        target_code = default_name if os.path.exists(default_name) else "template.py"
    print(f"Target: {task_id} ({url})")
    print(f"Script: {target_code}")
    samples = fetch_samples(url)
    if not samples:
        print("サンプルケースが見つかりませんでした。")
        sys.exit(1)
    print(f"{len(samples)} 件のサンプルを検出。テスト開始:\n")
    run_test(target_code, samples)
