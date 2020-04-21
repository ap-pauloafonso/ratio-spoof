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

class RatioSpoofState():
    def __init__(self, torrent_name, percent_complete, download_speed, upload_speed, announce_rate, current_downloaded, current_uploaded):
        self.torrent_name = torrent_name
        self.percent_complete = percent_complete
        self.download_speed = download_speed
        self.upload_speed = upload_speed
        self.announce_rate = announce_rate
        self.announce_history_deq = deque(maxlen=10)
        self.deq_count = 0
        self.announce_current_timer = self.announce_rate
        self.add_announce(current_downloaded, current_uploaded)
        threading.Thread(target = (lambda: self.__print_state())).start()

    def add_announce(self, current_downloaded, current_uploaded):
        self.deq_count +=1
        self.announce_history_deq.append({'count': self.deq_count, 'downloaded':current_downloaded,'uploaded':current_uploaded })
    def __decrease_timer(self):
        self.announce_current_timer = self.announce_current_timer - 1 if self.announce_current_timer > 0 else 0
    def reset_timer(self, new_announce_rate = None):
        if new_announce_rate != None:
            self.announce_rate = new_announce_rate
        self.announce_current_timer = self.announce_rate
    def __print_state(self):
        while True:
            print(f"""
###########################################################################
        Torrent: {self.torrent_name} - {self.percent_complete}% 
        download_speed: {self.download_speed}KB/s 
        upload_speed: {self.upload_speed}KB/s
###########################################################################
    """)
            for item in list(self.announce_history_deq)[:len(self.announce_history_deq)-1]:
                print(f'#{item["count"]} downloaded: {human_readable_size(item["downloaded"])}  | uploaded: {human_readable_size(item["uploaded"])} | announced')
            print(f'#{self.announce_history_deq[-1]["count"]} downloaded: {human_readable_size(self.announce_history_deq[-1]["downloaded"])}  | uploaded: {human_readable_size(self.announce_history_deq[-1]["uploaded"])} | next announce in :{str(datetime.timedelta(seconds=self.announce_current_timer))}')
            self.__decrease_timer()
            clear_screen()
            time.sleep(1) 


def t_total_size(data):
    return sum(map(lambda x : x['length'] , data['info']['files']))

def t_infohash_urlencoded(data, raw_data):
    info_offsets= data['info']['byte_offsets']
    info_bytes = hashlib.sha1(raw_data[info_offsets[0]:info_offsets[1]]).digest()
    return urllib.parse.quote(info_bytes).lower()

def t_piecesize_b(data):
    return data['info']['piece length']

def next_announce_total_b(kb_speed, b_current, b_piece_size,s_time, b_total_limit = None):
    total = b_current + (kb_speed *1024 *s_time)
    closest_piece_number = int(total / b_piece_size)
    closest_piece_number = closest_piece_number + random.randint(-10,10)
    next_announce = closest_piece_number *b_piece_size
    if(b_total_limit is not None and next_announce > b_total_limit):
        return b_total_limit    
    return next_announce

def next_announce_left_b(b_current, b_total_size):
    return b_total_size - b_current


def peer_id():
    peer_id =  f'-qB4030-{base64.urlsafe_b64encode(uuid.uuid4().bytes)[:12].decode()}'
    return peer_id

def find_approx_current(b_total_size, piece_size, percent):
    if( percent <= 0): return 0
    total = (percent/100) * b_total_size
    current_approx = int(total / piece_size) * piece_size
    return current_approx

def read_file(f, args_downalod, args_upload):
    raw_data = f.read()
    result = bencode_parser.decode(raw_data)
    piece_size = t_piecesize_b(result)
    total_size = t_total_size(result) 
    current_downloaded = find_approx_current(total_size,piece_size,args_downalod[0])
    current_uploaded = find_approx_current(total_size,piece_size,args_upload[0])
    state  = RatioSpoofState(result['info']['name'],args_downalod[0],args_downalod[1],args_upload[1],1800, current_downloaded, current_uploaded)
    
    while True:
         time.sleep(0.5)
"""     while True:
        if(current_downloaded < total_size):
            current_downloaded = next_announce_total_b(args_downalod[1],current_downloaded, piece_size, state.announce_rate, total_size)
        current_uploaded = next_announce_total_b(args_upload[1],current_uploaded, piece_size, state.announce_rate)
        state.add_announce(current_downloaded,current_uploaded)

        
        """

        


parser = argparse.ArgumentParser()
parser.add_argument('-t', required=True,help='path .torrent file' , type=argparse.FileType('rb'))
parser.add_argument('-d', required=True,type=int,help='parms for download', nargs=2 ,metavar=('%_COMPLETE', 'KBS_SPEED'))
parser.add_argument('-u',required=True,type=int,help='parms for upload', nargs=2 ,metavar=('%_COMPLETE', 'KBS_SPEED'))
args = parser.parse_args()
read_file(args.t, args.d, args.u)
