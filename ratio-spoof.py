import bencode_parser
import sys 
import hashlib
import urllib.parse
import json
import random 
import base64
import os
import uuid

def t_total_size(data):
    return sum(map(lambda x : x['length'] , data['info']['files']))

def t_infohash_urlencoded(data):
    info_offsets= result['info']['byte_offsets']
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

def next_announce_left_b(b_next_total, b_total_size):
    return b_total_size - b_next_total


def peer_id():
    peer_id =  f'-qB4030-{base64.urlsafe_b64encode(uuid.uuid4().bytes)[:12].decode()}'
    return peer_id

  



with open(sys.argv[1], 'rb') as f: 
    raw_data = f.read()
    result = bencode_parser.decode(raw_data)
    piece_size = t_piecesize_b(result)
    total_size = t_total_size(result)
    current  = piece_size
    while current < total_size:
        current = next_announce_total_b(50, current, piece_size, 1800, total_size)
        print(f'current: {current},  {int(current/piece_size)}/{int(total_size/piece_size)}')
    print(len(peer_id()))
   # print(t_infohash_urlencoded(result))
   
   # offsets =data['info']['byte_offsets']
    #info_hash = hashlib.sha1(raw_data[offsets[0]: offsets[1]]).hexdigest()
   # sha1_hash =hashlib.sha1(raw_data[offsets[0]: offsets[1]])
    #test  = hashlib.sha1(raw_data[offsets[0]: offsets[1]]).digest()
    #print(data['announce'])
    #print(urllib.parse.quote(test).lower())


