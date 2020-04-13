import socket
import struct
import sys
from enum import Enum
import os
import json
import hashlib
import os
import urllib.parse


class BencodeKeys(Enum):
    dic = 'd'
    number=  'i'
    arr = 'l'
    end_of_collecion = 'e'
    length_and_value_string_separator = ':'

    
def string_parse(startIdx, data:bytes):
    current  = startIdx
    while data[current:current +1].decode() != BencodeKeys.length_and_value_string_separator.value:
        current= current + 1
    size = data[startIdx:current].decode()
    string_nextidx = current+1 + int(size)
    data_slice = data[current+1:current+1 + int(size)]
    return (str(data_slice, 'utf-8', 'replace'), string_nextidx)

def number_parse(startIdx, data:bytes):
    current = startIdx
    while data[current:current +1].decode() != BencodeKeys.end_of_collecion.value:
           current = current +1
    number_nextidx = current +1
    data_slice = data[startIdx +1:current]
    return (int(data_slice), number_nextidx)

def find_parse(startIdx,data:bytes):
    c = data[startIdx:startIdx +1].decode()
    if(c == BencodeKeys.number.value):
        return number_parse(startIdx, data)
    elif(c == BencodeKeys.dic.value):
        return dic_parse(startIdx,data)
    elif(c == BencodeKeys.arr.value):
        return list_parse(startIdx,data)
    elif(str(c).isdigit()):   
        return string_parse(startIdx, data)
    else:
        raise Exception('Error parse')


def list_parse(startIdx, data):
    result = []
    current = startIdx +1
    while current < len(data):
        value, nextIdx = find_parse(current, data)
        result.append(value)
        current = nextIdx
        if (data[current: current+1].decode()== BencodeKeys.end_of_collecion.value):
           current = current +1
           break
    list_nextidx = current
    return (result, list_nextidx)


def dic_parse(startIdx,data):
    dic = {}
    initial_dict_idx = startIdx
    current = startIdx +1

    while current < len(data):
       key, nextIdx = find_parse(current, data)
       current = nextIdx
       value,nextIdx = find_parse(current, data)
       dic[key] = value
       current = nextIdx
       if (data[current: current+1].decode()==BencodeKeys.end_of_collecion.value):
           current = current +1
           final_dict_idx = current
           dic['byte_offsets'] =  [initial_dict_idx,final_dict_idx]
           break
    dic_nextidx = current
    return dic, dic_nextidx


def decode(data:bytes):
    result,_ = find_parse(0,data)
    return result





