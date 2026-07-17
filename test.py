import time
import requests
import urllib3
import concurrent.futures
# Suppress insecure certificate warnings
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
URL = "http://lms.signal.qlabs.pro:9000/login"
PAYLOAD = {"login": "student_test", "password": "123456"}
CONCURRENCY = 50
DURATION = 15
print(f"Starting load test on {URL} with concurrency={CONCURRENCY} for {DURATION} seconds...")
success_count = 0
fail_count = 0
latencies = []
def send_request():
    global success_count, fail_count
    start = time.time()
    try:
        r = requests.post(URL, json=PAYLOAD, verify=False, timeout=5)
        latency = time.time() - start
        if r.status_code == 200:
            success_count += 1
            latencies.append(latency)
        else:
            fail_count += 1
    except Exception as e:
        fail_count += 1
start_time = time.time()
with concurrent.futures.ThreadPoolExecutor(max_workers=CONCURRENCY) as executor:
    futures = []
    # Send requests continuously for DURATION seconds
    while time.time() - start_time < DURATION:
        # Keep the thread pool filled
        if len(futures) < CONCURRENCY:
            futures.append(executor.submit(send_request))
        else:
            # Clean up completed futures
            done, futures = concurrent.futures.wait(futures, return_when=concurrent.futures.FIRST_COMPLETED)
            futures = list(futures)
total_time = time.time() - start_time
total_requests = success_count + fail_count
rps = total_requests / total_time
avg_latency = sum(latencies) / len(latencies) if latencies else 0
print("\n--- Load Test Results ---")
print(f"Total Requests: {total_requests}")
print(f"Successful Logins: {success_count}")
print(f"Failed/Timeout Logins: {fail_count}")
print(f"Total Duration: {total_time:.2f} seconds")
print(f"Throughput (RPS): {rps:.2f}")
if latencies:
    print(f"Average Latency: {avg_latency*1000:.2f} ms")
    print(f"Min Latency: {min(latencies)*1000:.2f} ms")
    print(f"Max Latency: {max(latencies)*1000:.2f} ms")
