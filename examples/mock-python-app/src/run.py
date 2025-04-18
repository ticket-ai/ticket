"""
Startup script to run both the FastAPI server and the Mock LLM server.
"""

import subprocess
import sys
import time
import signal
import os

def main():
    """
    Start both the FastAPI server and the Mock LLM server.
    """
    print("Starting Guardian AI FastAPI Example...")
    
    # Start the Mock LLM server
    llm_process = subprocess.Popen(
        [sys.executable, "src/mock_llm_server.py"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1
    )
    print("Started Mock LLM server (PID: {})".format(llm_process.pid))
    
    # Give the LLM server a moment to start
    time.sleep(1)
    
    # Start the FastAPI server
    api_process = subprocess.Popen(
        [sys.executable, "src/server.py"],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1
    )
    print("Started FastAPI server (PID: {})".format(api_process.pid))
    
    # Setup signal handler for graceful shutdown
    def signal_handler(sig, frame):
        print("\nShutting down servers...")
        api_process.terminate()
        llm_process.terminate()
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    
    # Monitor output from both processes
    try:
        error_detected = False
        # Wait a moment to check for immediate errors
        time.sleep(2)
        
        # Check if either process has exited immediately
        if api_process.poll() is not None:
            print("ERROR: API server exited early with code:", api_process.returncode)
            error_detected = True
            # Get any error output
            api_stderr = api_process.stderr.read()
            api_stdout = api_process.stdout.read()
            print("API Server STDERR:", api_stderr)
            print("API Server STDOUT:", api_stdout)
        
        if llm_process.poll() is not None:
            print("ERROR: LLM server exited early with code:", llm_process.returncode)
            error_detected = True
            # Get any error output
            llm_stderr = llm_process.stderr.read()
            llm_stdout = llm_process.stdout.read()
            print("LLM Server STDERR:", llm_stderr)
            print("LLM Server STDOUT:", llm_stdout)
            
        if error_detected:
            print("One or more servers failed to start properly. Exiting.")
            signal_handler(None, None)
        
        while True:
            # Print API server output
            api_out = api_process.stdout.readline().strip()
            if api_out:
                print(f"[API] {api_out}")
                
            api_err = api_process.stderr.readline().strip()
            if api_err:
                print(f"[API ERROR] {api_err}")
            
            # Print LLM server output
            llm_out = llm_process.stdout.readline().strip()
            if llm_out:
                print(f"[LLM] {llm_out}")
                
            llm_err = llm_process.stderr.readline().strip()
            if llm_err:
                print(f"[LLM ERROR] {llm_err}")
            
            # Check if either process has exited
            if api_process.poll() is not None:
                print("API server has exited with code:", api_process.returncode)
                # Get any final error output
                api_stderr = api_process.stderr.read()
                if api_stderr:
                    print("API Server STDERR:", api_stderr)
                break
            
            if llm_process.poll() is not None:
                print("LLM server has exited with code:", llm_process.returncode)
                # Get any final error output
                llm_stderr = llm_process.stderr.read()
                if llm_stderr:
                    print("LLM Server STDERR:", llm_stderr)
                break
            
            # Small sleep to prevent high CPU usage
            time.sleep(0.1)
    except KeyboardInterrupt:
        print("\nShutting down servers...")
    finally:
        # Ensure both processes are terminated
        if api_process.poll() is None:
            api_process.terminate()
        if llm_process.poll() is None:
            llm_process.terminate()

if __name__ == "__main__":
    main()