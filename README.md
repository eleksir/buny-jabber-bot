# Buny Jabber Bot

Простой бот, работающий по протоколу xmpp. Его задача - банить абьюзеров чата.

Этот бот автоматизирует уборку человеческой грязи. Самый важный вопрос, который следует себе задать перед его
использованием - а не забаню ли я кого-то не того?

## Что он может?

Имеет возможность соединяться с сервером по протоколу xmpp как с шифрованием, так и без. (Пока что) Шифрование
работает по классическим механикам (шифрованный канал связи, без возможности коммуникации в незашифрованном виде),
без поддержки механизма start tls. Что-то в гошке или на стороне сервера при выборе StartTLS не работает.

Имеет возможность заносить пользователей в бан-лист, согласно заданным в чёрном списке правилам: по совпадению с
регулярными выражениями в nick-е или jid-е злодея. Правила можно настроимть как глобально, для всех комнат, где
присутсвует бот, так и для каждой комнаты отдельно. При изменении списка правил бота надо перезапускать.

## Что он не может?

* Не может заходить в комнаты, защищённые паролем.
* Не умеет разгадывать капчу.

Предполагается, что если на комнату навешен пароль, то такой бот там не нужен, а капча защищает от злодеев на 100%,
иначе её навешивать бесполезно.

## Как заставить его работать?

Технически, бот собирается гошкой 1.21 под платформу linux. Работоспособность на других платформах не тестировалась.

Чтобы получить готовые бинарники, достаточно выполнить команду:

```bash
make
```

в итоге получится бинарник **buny-jabber-bot**, которому нужен конфиг data/config.json, whitelist.json и blacklist.json
для работы.

В файлах **data/blacklist.json** и **data/whitelist.json** находятся списки злодеев и списки проверенных пользователей.

### Замечание

Очевидным образом в комнате банить может только админ. Поэтому боту придётся выдать админские права. А комнату сделать
не анонимной (то есть, чтобы как минимум админы могли видеть реальный jid, потому что в бан отправляется именно он).

Почему нельзя оставить комнату анонимной? А тут всё просто, в банлист заносится реальный jid пользователя, а не его ник.
Соответственно, бот должен видеть этот самый реальный jid. В анонимной комнате бот не сможет увидеть реальный jid.
А автокик - это конечно хорошо, но есть нюанс: можно, например, гадить ником.

## Как работает бан?

Никакой магии, всё по стандарту, согласно [xep-0045](https://xmpp.org/extensions/xep-0045.html#ban).

Серверу направляется команда изменения участия (affiliation) зашедшего jid-а и задаётся значение outcast, на что сервер
помещает этот jid в список изгнанных из комнаты (бан-лист конкретной комнаты, в которой присутствует бот) и выгоняет
этого jid-а.

### В чём тонкость?

Ровно в том, что для исполнения действия, боту надо заметить presence участника, а presence посылается
при изменении состояния клиента. То есть, если jid уже был в комнате на момент захода бота в неё, то его не забанят
(во всяком случае моментально). Самый простой способ поместить таких персонажей в бан - просто кикнуть их. Ихний
affiliation сменится на none, бот заметит presence и, если jid подпадает под правила чёрного списка, то отправит этот
jid в бан.

Но, простите, ведь "по правилам", при заходе клиента в комнату боту присылается presence всех участников, чтобы он мог
построить локальный ростер комнаты и начать отслеживать изменения состояния участников конференции. Почему бы в этот
момент не банить злодеев?

Да, так делать можно, однако пока этого нет. Дело в том, что presence самого себя участнику приходит последним (после
presence-ов тех, кто находится в комнате на момент появления в ней бота), и, согласно этому событию, можно считать, что
клиент зашёл в комнату успешно и формально *только после этого*, клиент может что-то пытаться делать в этой комнате.
Соответственно, чтобы кого-то банить исходя из списка presense-ов, который прилетает на входе, необходимо реализовать
"одноразовую" очередь тех, кого имеет смысл забанить и разгрести её "когда будет можно". Эта задача факультативная,
возможно, она будет решена потом.

## Как работает белый список?

Белый список - это список jid-ов, которых банить нельзя ни при каких обстоятельствах. В нём всегда неявно присутствует
сам бот.

### Можно ли почистить ban-list?

Да, бывают ситуации, когда по ошибке бот может забанить кого-то не того или, если не настроен белый список вообще ни в
каком виде, то будет банить всех входящих.

И если вдруг произошло так, что в бан попали все админы комнаты, то...

Во-первых, давать овнера боту не надо ни при каких обстоятельствах.

Во-вторых, разбанить их может только овнер, для чего надо зайти в комнату из-под аккаунта бота и разбанить всех кого
надо.

## Дисклеймер

Весь дисклеймер в файле LICENSE.txt :)

## Благодарности

Во-первых, `сообществу`, без него не было бы чятиков.

Во-вторых, `batman46(new)` из конференций на `jabber.ru`, за наблюдения и за некоторые идеи, воплощённые в этом боте.

В-третьих, `U2` из конференции `radio@conference.jabber.ru` за мысли, идеи и за позитивный настрой.
