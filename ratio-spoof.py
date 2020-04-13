import bencode_parser

with open('/home/pa/Downloads/TorrentDB-1586737303638.txt', 'rb') as f: 
    raw_data = f.read()
    result = bencode_parser.decode(raw_data)
    print(result)
   
   # offsets =data['info']['byte_offsets']
    #info_hash = hashlib.sha1(raw_data[offsets[0]: offsets[1]]).hexdigest()
   # sha1_hash =hashlib.sha1(raw_data[offsets[0]: offsets[1]])
    #test  = hashlib.sha1(raw_data[offsets[0]: offsets[1]]).digest()
    #print(data['announce'])
    #print(urllib.parse.quote(test).lower())