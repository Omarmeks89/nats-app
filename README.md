# nats-app

Тестовое задание этапа стажировки L0.

![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)

Реализовано:

* Подписка и получение сообщений из канала nats-streaming;
* Восстановление кеша из БД;
* Сохранение сообщений в кеш (in memory);
* Синхронизация состояния кеша с БД;
* Валидация входящих сообщений канала;
* HTTP endpoint для получения информации о заказе по id;

В каталоге `config` находятся конфигурационные файлы проекта.

## Библиотеки:

* `go-chi`      https://github.com/go-chi/chi;
* `pgx`         https://github.com/jackc/pgx;
* `validator`   https://github.com/go-playground/validator;
* `gcache`      https://github.com/bluele/gcache;
* `stan-server` github.com/nats-io/stan.go v0.10.4;

Ниже схема работы приложения:

