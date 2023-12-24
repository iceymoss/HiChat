create table user_basics
(
    id              bigint unsigned auto_increment
        primary key,
    created_at      datetime(3)               null,
    updated_at      datetime(3)               null,
    deleted_at      datetime(3)               null,
    name            longtext                  null,
    pass_word       longtext                  null,
    avatar          longtext                  null,
    gender          varchar(6) default 'male' null comment 'male表示男， famale表示女',
    phone           longtext                  null,
    email           longtext                  null,
    identity        longtext                  null,
    client_ip       longtext                  null,
    client_port     longtext                  null,
    salt            longtext                  null,
    login_time      datetime(3)               null,
    heart_beat_time datetime(3)               null,
    login_out_time  datetime(3)               null,
    is_login_out    tinyint(1)                null,
    device_info     longtext                  null
);

create index idx_user_basics_deleted_at
    on user_basics (deleted_at);

INSERT INTO hi_chat.user_basics (id, created_at, updated_at, deleted_at, name, pass_word, avatar, gender, phone, email, identity, client_ip, client_port, salt, login_time, heart_beat_time, login_out_time, is_login_out, device_info) VALUES (1, '2023-10-30 16:17:43.278', '2023-10-30 16:33:46.871', null, '小日子', '0192023a7bbd73250516f069df18b500$1269515130', './asset/upload/16986548221298498081.png', 'male', '', '', '5dbb815f191d811450739147eaddf0c2', '', '', '1269515130', '2023-10-30 16:17:43.175', '2023-10-30 16:17:43.175', '2023-10-30 16:17:43.175', 0, '');
INSERT INTO hi_chat.user_basics (id, created_at, updated_at, deleted_at, name, pass_word, avatar, gender, phone, email, identity, client_ip, client_port, salt, login_time, heart_beat_time, login_out_time, is_login_out, device_info) VALUES (2, '2023-10-30 16:34:19.176', '2023-10-30 16:34:19.176', null, 'iceymoss', '0192023a7bbd73250516f069df18b500$2019727887', '', 'male', '', '', '', '', '', '2019727887', '2023-10-30 16:34:19.176', '2023-10-30 16:34:19.176', '2023-10-30 16:34:19.176', 0, '');
INSERT INTO hi_chat.user_basics (id, created_at, updated_at, deleted_at, name, pass_word, avatar, gender, phone, email, identity, client_ip, client_port, salt, login_time, heart_beat_time, login_out_time, is_login_out, device_info) VALUES (3, '2023-10-30 16:34:50.279', '2023-10-30 16:34:58.661', null, 'yangkuang', '0192023a7bbd73250516f069df18b500$1427131847', '', 'male', '', '', '1f7d4acd9e1c452b9d8d7511b062a254', '', '', '1427131847', '2023-10-30 16:34:50.278', '2023-10-30 16:34:50.278', '2023-10-30 16:34:50.278', 0, '');


create table relations
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3)     null,
    updated_at datetime(3)     null,
    deleted_at datetime(3)     null,
    owner_id   bigint unsigned null,
    target_id  bigint unsigned null,
    type       bigint          null,
    `desc`     longtext        null
);

create index idx_relations_deleted_at
    on relations (deleted_at);

INSERT INTO hi_chat.relations (id, created_at, updated_at, deleted_at, owner_id, target_id, type, `desc`) VALUES (1, '2023-10-30 16:37:51.986', '2023-10-30 16:37:51.986', null, 3, 2, 1, '');
INSERT INTO hi_chat.relations (id, created_at, updated_at, deleted_at, owner_id, target_id, type, `desc`) VALUES (2, '2023-10-30 16:37:51.987', '2023-10-30 16:37:51.987', null, 2, 3, 1, '');
INSERT INTO hi_chat.relations (id, created_at, updated_at, deleted_at, owner_id, target_id, type, `desc`) VALUES (3, '2023-10-30 16:39:33.824', '2023-10-30 16:39:33.824', null, 1, 3, 1, '');
INSERT INTO hi_chat.relations (id, created_at, updated_at, deleted_at, owner_id, target_id, type, `desc`) VALUES (4, '2023-10-30 16:39:33.825', '2023-10-30 16:39:33.825', null, 3, 1, 1, '');
INSERT INTO hi_chat.relations (id, created_at, updated_at, deleted_at, owner_id, target_id, type, `desc`) VALUES (5, '2023-10-30 16:48:09.447', '2023-10-30 16:48:09.447', null, 1, 1, 2, '');



create table messages
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3) null,
    updated_at datetime(3) null,
    deleted_at datetime(3) null,
    form_id    bigint      null,
    target_id  bigint      null,
    type       bigint      null,
    media      bigint      null,
    content    longtext    null,
    pic        longtext    null,
    url        longtext    null,
    `desc`     longtext    null,
    amount     bigint      null
);

create index idx_messages_deleted_at
    on messages (deleted_at);

INSERT INTO hi_chat.messages (id, created_at, updated_at, deleted_at, form_id, target_id, type, media, content, pic, url, `desc`, amount) VALUES (1, '2023-10-30 17:03:43.065', '2023-10-30 17:03:43.065', null, 1, 3, 1, 1, '吃了', '', '', '', 0);
INSERT INTO hi_chat.messages (id, created_at, updated_at, deleted_at, form_id, target_id, type, media, content, pic, url, `desc`, amount) VALUES (2, '2023-10-30 17:04:07.065', '2023-10-30 17:04:07.065', null, 3, 1, 1, 1, '滚', '', '', '', 0);



create table group_infos
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3)     null,
    updated_at datetime(3)     null,
    deleted_at datetime(3)     null,
    name       longtext        null,
    owner_id   bigint unsigned null,
    type       bigint          null,
    icon       longtext        null,
    `desc`     longtext        null
);

create index idx_group_infos_deleted_at
    on group_infos (deleted_at);



create table communities
(
    id         bigint unsigned auto_increment
        primary key,
    created_at datetime(3)     null,
    updated_at datetime(3)     null,
    deleted_at datetime(3)     null,
    name       longtext        null,
    owner_id   bigint unsigned null,
    type       bigint          null,
    image      longtext        null,
    `desc`     longtext        null
);

create index idx_communities_deleted_at
    on communities (deleted_at);

INSERT INTO hi_chat.communities (id, created_at, updated_at, deleted_at, name, owner_id, type, image, `desc`) VALUES (1, '2023-10-30 16:48:09.446', '2023-10-30 16:48:09.446', null, '涩涩圈', 1, 1, './asset/upload/16986556681474941318.jpeg', '你懂的！');

