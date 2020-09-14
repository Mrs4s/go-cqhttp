<?php
header('Content-Type=text/html;Charset=utf-8');
include  'config.php';
//开启报错信息
define('APP_DEBUG', false);
//项目名称
define('APP_NAME','shop');
//项目的目录
define('APP_PATH','application/default/');
//root目录
//define('__ROOT__','/vps');
//define('__APP__','');
//使用thinkPHP框架
include_once '../../library/ThinkPHP/ThinkPHP.php';
