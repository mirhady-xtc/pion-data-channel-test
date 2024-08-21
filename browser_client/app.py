
from flask import Flask, render_template, send_from_directory, request, jsonify
import os
import sqlalchemy
import httpagentparser
import sys
import signal
import argparse
from waitress import serve
import json
# other necessary imports for database

app = Flask(__name__, static_folder='.', static_url_path='')

USE_HTTPS = True

@app.route('/')
def index():
    # Read the content of the original HTML file
    with open('index.html', 'r') as f:
        html_content = f.read()
    
    # Replace exp_id with 3
    # html_content = html_content.replace('var exp_id = -1;', 'var exp_id = 3;')
    html_content = html_content.replace('var num_paths = -1;', f'var num_paths = {NUM_PATHS};')
    
    # Serve the modified HTML content
    return html_content

def parse_user_agent(user_agent_string):
    return httpagentparser.detect(user_agent_string)

@app.route('/save_system_info', methods=['POST'])
def save_system_info():

    # get os, platform, browser info
    user_agent = request.headers.get('User-Agent')
    parsed_info = parse_user_agent(user_agent)
    print(f"User-Agent Info: {parsed_info}")

    # get ip address
    user_ip = request.remote_addr
    print(f"User IP Address: {user_ip}")

    # data from js
    data = request.json
    print(f"Screen Resolution: {data.get('width')} x {data.get('height')}")
    print(f"Browser Window Size: {data.get('windowWidth')} x {data.get('windowHeight')}")
    print(f"Timezone: {data.get('timezone')}")
    print(f"Language: {data.get('language')}")
    print(f"Connection Type: {data.get('e_type')}")
    print(f"Device Memory: {data.get('deviceMemory')} GB")
    print(f"Hardware Concurrency: {data.get('hardwareConcurrency')}")

    metadata = {
        "host": HOST,
        "port": PORT,
        "download_required": DOWNLOAD_REQUIRED
    }

    response_data = {
        "status": "success",
        "system_info": parsed_info,
        "metadata": metadata
    }

    return jsonify(response_data)

@app.route('/receive_data', methods=['POST'])
def receive_data():
    # Get JSON data sent from the client
    data = request.json
    print("Data received:", data)

    # Perform any processing with the data here

    # Return a response
    return jsonify({"status": "success", "message": "Data received successfully"})
    

def signal_handler(sig, frame):
    print('quitting')
    sys.exit(0)


HOST = None
PORT = None
NUM_PATHS = None    


if __name__ == '__main__':

    # collect commandline arguments
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="0.0.0.0", help="Host IP address for the flask app")
    parser.add_argument("--port", default=8082, help="Port for the flask app")
    parser.add_argument("--download", action="store_true", help="Flag to indicate if a download is required")
    parser.add_argument("--num_paths", default=2, help="Number of paths to be used")
    args=parser.parse_args()

    HOST = args.host
    PORT = args.port
    DOWNLOAD_REQUIRED = args.download
    NUM_PATHS = args.num_paths

    # setup handler for terminal exit
    signal.signal(signal.SIGINT, signal_handler)

    serve(app, host=HOST, port=PORT, url_scheme='https')

