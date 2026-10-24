"""Scripted telnet driver: sends steps, logs raw server bytes (ANSI kept)."""
import socket, sys, time, json, select
IAC=255
def strip_telnet(b, sock):
    out=bytearray(); i=0
    while i < len(b):
        c=b[i]
        if c==IAC and i+1 < len(b):
            cmd=b[i+1]
            if cmd in (251,252,253,254) and i+2 < len(b):
                opt=b[i+2]
                # refuse everything except echo-related negotiation
                if cmd==253: sock.send(bytes([IAC,252,opt]))
                elif cmd==251: sock.send(bytes([IAC,254 if opt!=1 else 253,opt]))
                i+=3; continue
            if cmd==IAC: out.append(IAC); i+=2; continue
            i+=2; continue
        out.append(c); i+=1
    return bytes(out)
def run(port, steps, log):
    s=socket.create_connection(('127.0.0.1',port))
    events=[]; t0=time.time()
    def pump(dur):
        end=time.time()+dur
        while True:
            left=end-time.time()
            if left<=0: break
            r,_,_=select.select([s],[],[],left)
            if r:
                d=s.recv(65536)
                if not d: return False
                d=strip_telnet(d,s)
                if d: events.append({'t':round(time.time()-t0,2),'out':d.decode('latin-1')})
        return True
    for st in steps:
        cmd, wait = st[0], st[1]
        if cmd is not None:
            events.append({'t':round(time.time()-t0,2),'in':cmd})
            s.send((cmd+'\r\n').encode())
        if not pump(wait): break
    s.close()
    json.dump(events, open(log,'w'), indent=0)
if __name__=='__main__':
    port=int(sys.argv[1]); steps=json.load(open(sys.argv[2])); run(port, steps, sys.argv[3])
