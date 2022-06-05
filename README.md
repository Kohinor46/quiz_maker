# quiz_maker

Этот телеграм бот создаст вам quiz игру по правилам "Кто хочет стать миллионерам?", с одним правилом, что не правильные ответы не выводят игроков из игры

Для работы бота необходимо заполнить конфиг файл по примеру config_example.yaml

Обезятельные поля:

Path_to_files - переменная где бот будет сохранят все действия и искать файлы

Telegram_token - токен телеграм бота создается в @BotFather

Welcome - текст приветсвия (то тчо будет отправлено после /start)

Admin_ids - id ведущего/админа (он запускает раунды, проверяет результаты)

Rounds - раунды со следующим наполнением:
    
    Queston - вопрос
    Answers - варианты ответов
    Right_answer - правильный ответ
    Points - количество очков за раунд
    Fifty_fifty_buttons - кнопки для использования подсказки 50 на 50
    
Не обязательные поля:

Welcome_with_photo, Welcome_with_video, - переменная для отправки картинки с привествием или видео

Welcome_photo_from_disk,Welcome_photo_from_url,Welcome_video_from_disk,Welcome_video_from_url - от куда брать материал (диск или ссылка)

Media - ссылка или путь до файла на диске

В Rounds тоже есть необязетельыне поля:

    With_photo, With_video, With_audio - если требуется отправить картинку, видео или аудиофайл в раунде
    
    Photo_from_disk, Photo_from_url, Video_from_disk, Video_from_url, With_audio , Audio_from_disk, Audio_from_url - от куда брать матеиал (диск или ссылка)
    
    Media  - ссылка или путь до файла на диске
