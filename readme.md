我想要构架一个类似于王者荣耀的匹配系统，每个user有一个分数，这个分数在0-3000范围内，
客户端有几个操作支持
Register：注册一个用户，这个时候给他一个随机的分数，参数有用户名，密码，用户名不允许一致
Login：登陆，参数是用户名密码
Match：注册到匹配系统，如果成功返回匹配成功的一局游戏的token，

设计几张sql表格，
User表格，保存username， password和分数
match表格，username，分数，是否已经匹配完成。
game表格，一个token和这局游戏里面的十位玩家的id

redis里面username和对应的token，在match表格和game表格里面user用token的形式显示。
还要存每局游戏对应的token和状态

kafka负责消息发布。