port: 7880
rtc:
    #node_ip: {{ ansible_default_ipv4.address }}
    udp_port: 7882
    #tcp_port: 7881
    #tcp_port: 0
    use_external_ip: true
    #use_ice_lite: true
    use_ice_lite: false
    #port_range_start: 35200
    #port_range_end: 35400
    #force_tcp: true
    allow_tcp_fallback: false

#room:
#    enabled_codecs:
#        - mime: audio/opus
#        - mime: video/h264

turn:
    enabled: true
    udp_port: 3478

redis:
    address: redis:6379
    username: ""
    password: ""
    db: 2
keys:
    APIznJxWShGW3Kt: KEUUtCDVRqXk9me0Ok94g8G9xwtnjMeUxfNMy8dow6iA
logging:
  json: false
  level: info

webhook:
  api_key: 'APIznJxWShGW3Kt'
  urls:
    - 'http://video:1237/internal/livekit-webhook'
