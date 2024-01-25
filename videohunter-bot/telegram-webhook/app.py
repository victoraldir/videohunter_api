import os
import re
import json
import urllib.request

headers = {"Content-Type": "application/json"}

# Get token from environment variable.
telegram_bot_token = os.environ['BOT_TOKEN']

def is_valid_url(url):
        regex = re.compile(
            r'^(?:http|ftp)s?://'  # http:// ou https://
            r'(?:(?:[A-Z0-9](?:[A-Z0-9-]{0,61}[A-Z0-9])?\.)+(?:[A-Z]{2,6}\.?|[A-Z0-9-]{2,}\.?)|'  # domínio
            r'localhost|'  # localhost...
            r'\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|'  # ...ou endereço IP
            r'\[?[A-F0-9]*:[A-F0-9:]+\]?)'  # ...ou endereço IPv6
            r'(?::\d+)?'  # porta
            r'(?:/?|[/?]\S+)$', re.IGNORECASE
        )
        return bool(regex.match(url))

def send_message(message:str, chat_id:str):
    telegram_url = f'https://api.telegram.org/bot{telegram_bot_token}/sendMessage'
    params = {
        'chat_id': chat_id,
        'text': message,
    }

    print('######')
    print("message: ", params)
    print('######')

    print('######')
    print("url: ", telegram_url)
    print('######')

    body_message = json.dumps(params).encode('utf-8')
    request = urllib.request.Request(telegram_url, headers=headers, data=body_message, method='POST')
    response = urllib.request.urlopen(request, timeout=5)


def lambda_handler(event, context):
    """Sample pure Lambda function

    Parameters
    ----------
    event: dict, required
        API Gateway Lambda Proxy Input Format

        Event doc: https://docs.aws.amazon.com/apigateway/latest/developerguide/set-up-lambda-proxy-integrations.html#api-gateway-simple-proxy-for-lambda-input-format

    context: object, required
        Lambda Context runtime methods and attributes

        Context doc: https://docs.aws.amazon.com/lambda/latest/dg/python-context-object.html

    Returns
    ------
    API Gateway Lambda Proxy Output Format: dict

        Return doc: https://docs.aws.amazon.com/apigateway/latest/developerguide/set-up-lambda-proxy-integrations.html
    """

    url = "https://692d-2804-7f7-a6b9-9f28-b84c-ba1c-cb1-7c0.ngrok-free.app/url"

    body = json.loads(event['body'])

    print('######')
    print(body)
    print('######')
    
    if not body or not body['message']:
        return {
            "statusCode": 200,
            "body": "ok",
        }
    
    telegram_chat_id = body['message']['from']['id']

    if not is_valid_url(body['message']['text']):
        send_message("Url inválida, envie uma url válida do twitter.", telegram_chat_id)
        return {
            "statusCode": 200,
            "body": "ok",
        }
         

    url_twitter = body['message']['text']

    data = {"video_url": url_twitter}
    data = json.dumps(data).encode('utf-8')

    print('######')
    print("data video hunter", data)
    print('######')

    request = urllib.request.Request(url, headers=headers, data=data, method='POST')
    response = urllib.request.urlopen(request, timeout=5)

    resposta_json = json.loads(response.read().decode('utf-8'))
    video_id = resposta_json['id']

    print('######')
    print("video id: ", video_id)
    print('######')

    full_message = f"Segue a url do video: \n\n {url}/{video_id}"    

    send_message(full_message, telegram_chat_id)

    return {
        "statusCode": 200,
        "body": "ok",
    }