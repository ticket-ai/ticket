"""
Command-line interface for Guardian AI
"""

import argparse
import sys
import time
import signal
from . import Guardian, __version__


def main():
    """
    Main entry point for the Guardian AI command-line interface.
    """
    parser = argparse.ArgumentParser(
        description="Guardian AI - Ethical AI Monitoring and Governance Platform"
    )
    
    parser.add_argument(
        "--version", action="store_true",
        help="Show version information"
    )
    
    parser.add_argument(
        "--port", type=int,
        help="Port to listen on (default: auto-select)"
    )
    
    parser.add_argument(
        "--service", type=str, default="guardian-app",
        help="Service name (default: guardian-app)"
    )
    
    parser.add_argument(
        "--env", type=str, default="development",
        help="Environment (development, staging, production)"
    )
    
    parser.add_argument(
        "--pre-prompt", type=str,
        help="Standard pre-prompt to apply to all requests"
    )
    
    parser.add_argument(
        "--rules", type=str,
        help="Path to guardian_rules.json file"
    )
    
    parser.add_argument(
        "--debug", action="store_true",
        help="Enable debug mode"
    )
    
    args = parser.parse_args()
    
    # Show version and exit if requested
    if args.version:
        print(f"Guardian AI v{__version__}")
        return 0
    
    # Create guardian config
    config = {
        "service_name": args.service,
        "environment": args.env,
        "debug": args.debug,
    }
    
    if args.port:
        config["port"] = args.port
    
    if args.pre_prompt:
        config["pre_prompt"] = args.pre_prompt
    
    # Start Guardian
    try:
        # Create the Guardian instance (it will auto-start by default)
        guardian = Guardian(config)
        
        print(f"Guardian AI v{__version__} proxy server started on port {guardian.port}")
        print("Press Ctrl+C to exit")
        
        # Set up signal handler for graceful shutdown
        def signal_handler(sig, frame):
            print("\nShutting down Guardian AI...")
            guardian.stop()
            print("Guardian AI stopped.")
            sys.exit(0)
        
        signal.signal(signal.SIGINT, signal_handler)
        signal.signal(signal.SIGTERM, signal_handler)
        
        # Keep the process running
        try:
            while True:
                time.sleep(1)
        except KeyboardInterrupt:
            print("\nShutting down Guardian AI...")
            guardian.stop()
            print("Guardian AI stopped.")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    
    return 0


if __name__ == "__main__":
    sys.exit(main())