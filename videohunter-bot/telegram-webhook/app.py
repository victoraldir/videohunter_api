import os
import re
import json
import urllib.request
import urllib.error
import base64

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

    print('######')
    print("response: ", response)
    print('######')



def lambda_handler(event, context):

    print('######')
    print("event: ", event)
    print('######')

    try:
        url = "https://myvideohunter.com/prod/url"

        # Decode base64 body
        # bodyDecoded = base64.b64decode(event['body'])

        print('######')
        # print("bodyDecoded: ", bodyDecoded)
        print('######')

        body = json.loads(event['body'])
        
        try:
            if not body or not body['message']:
                print('######')
                print("Key 'message' not found in the body.")
                print('######')
                return {
                    "statusCode": 200,
                    "body": "ok",
                }
        except KeyError:
            print('######')
            print("Key 'message' not found in the body.")
            print('######')
            return {
                "statusCode": 200,
                "body": "Key 'message' not found in the body",
            }
        
        telegram_chat_id = body['message']['from']['id']

        if not is_valid_url(body['message']['text']):
            print('######')
            print("Invalid URL")
            print('######')
            send_message("Invalid URL", telegram_chat_id)
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

        try:
            request = urllib.request.Request(url, headers=headers, data=data, method='POST')
            response = urllib.request.urlopen(request, timeout=5)
        except urllib.error.HTTPError as e:
            print('######')
            print('HTTPError: ', e.code)
            print('Error message: ', e.read())
            print('######')
            return {
                "statusCode": 200,
                "body": "HTTP Error occurred",
            }

        resposta_json = json.loads(response.read().decode('utf-8'))
        video_id = resposta_json['id']

        print('######')
        print("video id: ", video_id)
        print('######')

        full_message = f"Here's you video: \n\n {url}/{video_id}"    

        send_message(full_message, telegram_chat_id)
    except Exception as e:
        print('######')
        print("Error: ", e)
        print('######')
        # send_message("An error occurred", telegram_chat_id)

    # Return a 200 status code to the Telegram API anyways 
    return {
        "statusCode": 200,
        "body": "ok",
    }