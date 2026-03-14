create database if not exists gamequest;
use gamequest;

create table if not exists users (
    Id int auto_increment primary key,
    UserName varchar(255) not null unique,
    Password varchar(255) not null,
    Score int not null
);
create table match_queue (
    Id int unique not null,
    Score int not null,
    MatchId int unique not null
); 
create table match_history  (
    Id int unique not null, 
    Score int not null,
    GameId int not null, 
    MatchId int unique not null
);
create table game_info (
    GameId int auto_increment primary key,
    MaxScore int not null,
    MinScore int not null
);
create table game_players (
    GameId int not null,
    PlayerId int not null,
    Score int not null
);
