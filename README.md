# Today's Duty Bot

## Setup Test MySQL

```console
$ docker run -d --name today-duty-bot-mysql -e MYSQL_ALLOW_EMPTY_PASSWORD=yes -p 3306:3306 --health-cmd "mysqladmin ping -h 127.0.0.1" mysql:8
$ mysql -u root -h 127.0.0.1 < ddl/database.sql
$ mysql -u root -h 127.0.0.1 bot < ddl/schema.sql
```

## License

[MIT](LICENSE)

## Author

Masahiro Furudate (a.k.a. [178inaba](https://github.com/178inaba))  
<178inaba.git@gmail.com>
