#!/usr/bin/env python3
import bencode_parser
import sys 
import hashlib
import urllib.parse
import json
import random 
import base64
import os
import uuid
import argparse
import time
from collections import deque
import subprocess
import platform
import datetime
import threading
import urllib.request
import http.client
import gzip



class RatioSpoofState():
    def __init__(self, torrent_name,download_speed, upload_speed, \
                    announce_rate, current_downloaded, current_uploaded,\
                    piece_size, total_size, announce_info, info_hash_urlencoded):
        self.__lock =  threading.Lock()
        self.torrent_name = torrent_name
        self.percent_complete = round(current_downloaded / total_size)
        self.download_speed = download_speed
        self.upload_speed = upload_speed
        self.announce_rate = announce_rate
        self.announce_current_timer = self.announce_rate
        self.piece_size = piece_size
        self.total_size = total_size
        self.announce_info = announce_info
        self.peer_id = peer_id()
        self.key = key()
        self.info_hash_urlencoded = info_hash_urlencoded
        self.announce_history_deq = deque(maxlen=10)
        self.deq_count = 0
        self.numwant = 200
        self.__add_announce(current_downloaded, current_uploaded ,next_announce_left_b(current_downloaded, total_size))
        threading.Thread(daemon = True, target = (lambda: self.__print_state())).start()
        threading.Thread(daemon = True, target = (lambda: self.__decrease_timer())).start()

    def start_announcing(self):
        announce_rate = self.__announce('started')
        while True:
            self.__generate_next_announce(announce_rate)
            time.sleep(announce_rate)
            self.__announce()

    def __add_announce(self, current_downloaded, current_uploaded, left):
        self.deq_count +=1
        self.announce_history_deq.append({'count': self.deq_count, 'downloaded':current_downloaded, 'percent': round((current_downloaded/self.total_size) *100) , 'uploaded':current_uploaded,'left': left })
    
    def __generate_next_announce(self, announce_rate):
        self.__reset_timer(announce_rate)
        current_downloaded = self.announce_history_deq[-1]['downloaded'] 
        if(self.announce_history_deq[-1]['downloaded'] < self.total_size):
            current_downloaded = next_announce_total_b(self.download_speed,self.announce_history_deq[-1]['downloaded'], self.piece_size, self.announce_rate, self.total_size)
        else: 
            self.numwant = 0
        current_uploaded = next_announce_total_b(self.upload_speed,self.announce_history_deq[-1]['uploaded'], self.piece_size, self.announce_rate)
        current_left  = next_announce_left_b(current_downloaded, self.total_size)
        self.__add_announce(current_downloaded,current_uploaded,current_left)
    
    def __announce(self, event = None):
        last_announce_data = self.announce_history_deq[-1]
        query_dict  = build_query_string(self, last_announce_data, event)

        error =''

        if (len(self.announce_info['list_of_lists']) > 0):
            for tier_list in self.announce_info['list_of_lists']:
                for item in tier_list:
                    try:
                        return tracker_announce_request(item, query_dict)
                    except Exception as e : error = str(e)
                
        else:
            url = self.announce_info['main']
            try:
                return tracker_announce_request(url, query_dict)
            except Exception as e : error = str(e)
            
        raise Exception(f'Connection error with the tracker: {error}')

    def __decrease_timer(self):
        while True:
            time.sleep(1)
            with self.__lock:
                self.announce_current_timer = self.announce_current_timer - 1 if self.announce_current_timer > 0 else 0

    def __reset_timer(self, new_announce_rate = None):
        if new_announce_rate != None:
            self.announce_rate = new_announce_rate
        with self.__lock:
            self.announce_current_timer = self.announce_rate
    
    def __print_state(self):
        while True:
            print(f"""
            ###########################################################################
            Torrent: {self.torrent_name} - {self.percent_complete}% 
            download_speed: {self.download_speed}KB/s 
            upload_speed: {self.upload_speed}KB/s
            size: {human_readable_size(self.total_size)}
            ###########################################################################
            """)
            for item in list(self.announce_history_deq)[:len(self.announce_history_deq)-1]:
                print(f'#{item["count"]} downloaded: {human_readable_size(item["downloaded"])} ({item["percent"]}%)| left: {human_readable_size(item["left"])}  | uploaded: {human_readable_size(item["uploaded"])} | announced')
            print(f'#{self.announce_history_deq[-1]["count"]} downloaded: {human_readable_size(self.announce_history_deq[-1]["downloaded"])} ({self.announce_history_deq[-1]["percent"]}%) | left: {human_readable_size(self.announce_history_deq[-1]["left"])}   | uploaded: {human_readable_size(self.announce_history_deq[-1]["uploaded"])} | next announce in :{str(datetime.timedelta(seconds=self.announce_current_timer))}')
            clear_screen()
            time.sleep(1)


def human_readable_size(size, decimal_places=2):
    for unit in ['B','KiB','MiB','GiB','TiB']:
        if size < 1024.0:
            break
        size /= 1024.0
    return f"{size:.{decimal_places}f}{unit}"

def clear_screen():
    if platform.system() == "Windows":
        subprocess.Popen("cls", shell=True).communicate()
    else:
        print("\033c", end="")

def t_total_size(data):
    return sum(map(lambda x : x['length'] , data['info']['files']))

def t_infohash_urlencoded(data, raw_data):
    info_offsets= data['info']['byte_offsets']
    info_bytes = hashlib.sha1(raw_data[info_offsets[0]:info_offsets[1]]).digest()
    return urllib.parse.quote_plus(info_bytes)

def t_piecesize_b(data):
    return data['info']['piece length']

def next_announce_total_b(speed_kbps, b_current, b_piece_size,s_time, b_total_limit = None):
    if(speed_kbps == 0): return b_current
    
    total = b_current + (speed_kbps *1024 *s_time)
    closest_piece_number = int(total / b_piece_size)
    closest_piece_number = closest_piece_number + random.randint(1,10)
    next_announce = closest_piece_number *b_piece_size

    if(b_total_limit is not None and next_announce > b_total_limit):
        return b_total_limit    
    return next_announce

def next_announce_left_b(b_current, b_total_size):
    return b_total_size - b_current

def peer_id(): 
    return f'-qB4030-{base64.urlsafe_b64encode(uuid.uuid4().bytes)[:12].decode()}'

def key():
    return hex(random.getrandbits(32))[2:].upper()

def find_approx_current(b_total_size, piece_size, percent):
    if( percent <= 0): return 0
    total = (percent/100) * b_total_size
    current_approx = int(total / piece_size) * piece_size
    return current_approx

def build_announce_info(data):
    announce_info = {'main':data['announce'], 'list_of_lists':data['announce-list'] if 'announce-list' in data else []}
    tcp_list_of_lists = []
    for _list in announce_info['list_of_lists']:
        aux = list(filter(lambda x: x.lower().startswith('http'),_list))
        if len(aux) >0:
            tcp_list_of_lists.append(aux) 
    announce_info['list_of_lists'] = tcp_list_of_lists
    if (not announce_info['main'].startswith('udp')):
        announce_info['list_of_lists'].insert(-1,[announce_info['main']])

    if(len(announce_info['list_of_lists']) == 0): raise Exception('NO tcp/http tracker url announce found')

    return announce_info

def tracker_announce_request(url, query_string):
    request = urllib.request.Request(url = f'{url}?{query_string}', headers= {'User-Agent' :'qBittorrent/4.0.3', 'Accept-Encoding':'gzip'})	
    response = urllib.request.urlopen(request).read()
    try:
        response = gzip.decompress(response)
    except:pass

    decoded_response  =  bencode_parser.decode(response)
    
    interval = decoded_response.get('min interval',None)
    if(interval is None):
        interval = decoded_response.get('interval',None)

    if 'interval' is not None:
        return int(decoded_response['interval'])
    else: raise Exception(json.dumps(decoded_response))

def build_query_string(state:RatioSpoofState, curent_info, event):    
    query = {
    'peer_id':state.peer_id,
    'port':8999,
    'uploaded':curent_info['uploaded'],
    'downloaded':curent_info['downloaded'],
    'left':curent_info['left'],
    'corrupt': 0,
    'key':state.key,
    'event':event,
    'numwant':state.numwant,
    'compact':1,
    'no_peer_id': 1,
    'supportcrypto':1,
    'redundant':0
    }

    if(event == None):
        del(query['event'])

    result = f'info_hash={state.info_hash_urlencoded}&' + urllib.parse.urlencode(query)
    return result

def check_initial_value_suffix(input:str, attribute_name):
    valid_suffixs = ('%', 'b','kb','mb','gb','tb')
    if not input.lower().endswith(valid_suffixs):
        raise Exception(f'initial {attribute_name} must be in {valid_suffixs}')

def check_speed_value_suffix(input:str, attribute_name):
    valid_suffixs = ('kbps')
    if not input.lower().endswith(valid_suffixs):
        raise Exception(f'{attribute_name} speed must be in {valid_suffixs}')

def percent_validation(n):
    if n not in range (0, 101):
        raise Exception ('percent value must be in (0-100)')

def input_size_2_byte_size(input, total_size ):        
    if input.lower().endswith('kb'):
        return (int(input[:-2])) * 1024
    elif input.lower().endswith('mb'):
        return (int(input[:-2])) * (1024 **2)
    elif input.lower().endswith('gb'):
        return (int(input[:-2])) * (1024 **3)
    elif input.lower().endswith('tb'):
        return (int(input[:-2])) * (1024 **4)
    elif input.lower().endswith('b'):
        return int(input[:-1])
    elif input.endswith('%'):
        percent_validation(int(input[:-1]))
        return int((int(input[:-1])/100 ) * total_size)

def check_downloaded_initial_value(input, total_size_b):
    size_b  =input_size_2_byte_size(input,total_size_b)
    if size_b > total_size_b:
        raise Exception('initial downloaded can not be higher than the torrent size')
    return size_b

def check_uploaded_initial_value(input, total_size_b):
    size_b  =input_size_2_byte_size(input, total_size_b)
    return size_b

def check_speed(input):
    return int(input[:-4])


def validate_download_args(downloaded_arg, download_speed_arg,total_size_b):
    check_initial_value_suffix(downloaded_arg,'download')
    check_speed_value_suffix(download_speed_arg, 'download')
    
    donwloaded_b = check_downloaded_initial_value(downloaded_arg, total_size_b)
    speed_kbps = check_speed(download_speed_arg)

    return (donwloaded_b, speed_kbps)

def validate_upload_args(uploaded_arg, upload_speed_arg, total_size_b):
    check_initial_value_suffix(uploaded_arg,'upload')
    check_speed_value_suffix(upload_speed_arg, 'upload')
    
    uploaded_b = check_uploaded_initial_value(uploaded_arg,total_size_b )
    speed_kbps = check_speed(upload_speed_arg)
    return (uploaded_b, speed_kbps)


def read_file(f, args_download, args_upload):
    raw_data = f.read()
    result = bencode_parser.decode(raw_data)
    total_size = t_total_size(result)
    piece_size = t_piecesize_b(result)

    downloaded, download_speed_kbps = validate_download_args(args_download[0], args_download[1], total_size)
    uploaded_b, upload_speed_kbps = validate_upload_args(args_upload[0], args_upload[1], total_size)

    state  = RatioSpoofState(result['info']['name'],download_speed_kbps,upload_speed_kbps,0,\
                             downloaded, uploaded_b, piece_size,total_size,
                             build_announce_info(result),t_infohash_urlencoded(result, raw_data))

    state.start_announcing()

        
parser = argparse.ArgumentParser()
parser.add_argument('-t', required=True, metavar=('(TORRENT_PATH)'), help='path .torrent file' , type=argparse.FileType('rb'))
parser.add_argument('-d', required=True,help='parms for download', nargs=2 ,metavar=('(%_COMPLETE)', '(KBS_SPEED)'))
parser.add_argument('-u',required=True,help='parms for upload', nargs=2 ,metavar=('(%_COMPLETE)', '(KBS_SPEED)'))
args = parser.parse_args()

read_file(args.t, args.d, args.u)
