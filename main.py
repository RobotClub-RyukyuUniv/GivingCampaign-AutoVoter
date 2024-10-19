"""
このスクリプトは、指定されたURLのフォームを自動で入力し、送信するためのものです。
主な仕様:
- Seleniumを使用してブラウザを自動操作します。
- フォームの各フィールドに指定された情報を入力します。
- フォームを送信します。

定数の説明
URL: それぞれ大学の参加団体を選択したところ（必ずドメイン以降は/form/vote/step1であること)
TARGET: 入力したい団体名。

NAME: 名前（半角スペースで区切る）
GENDER: 性別（男、女、その他、無回答）
BIRTHDAY: 誕生日。/で区切る。例"2005/12/31"
EMAIL: メールアドレス
PHONE_NUMBER: 電話番号
RELATION = "他大学の学部生"
MYUNIVERSITY = "琉球大学"
ENCROLLMENT: 入学年月（例: "2024/4")
GRADUATION: 卒業年月（例: "2028/3")

MAJOR: 理系か文系かその他か、理系："science", 文系: "humanities", その他: "other"
MAJORSUBJECT: 学部学科

MOTIVE: 投票理由    以前から交流会: "relationship_with_individual_or_organization"  きっかけに交流: "active_individual_or_organization_relationship" 

HOWTOKNOW:どうやって知った 学生から:"student"   弊学の学生以外の個人からの紹介: "other_individual"   SNSから:"university_official_announcement"   Giving Campaign運営による発信（プレスリリース・公式SNSなど）: "giving_campaign_official_announcement"
"""

from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait, Select
from selenium.webdriver.support import expected_conditions as EC
import time

#URL = 'https://hirosaki-u.2024.giving-campaign.jp/form/vote/step1'

#TARGET ="弘前大学書道部"

NAME = "宮城 琉徳"
GENDER = "無回答"
BIRTHDAY = "2005/12/31"
EMAIL = "e245719@ie.u-ryukyu.ac.jp"
PHONE_NUMBER = "08064846447"
RELATION = "他大学の学部生"
MYUNIVERSITY = "琉球大学"
ENCROLLMENT = "2024/4"
GRADUATION = "2028/3"
MAJOR = "science"
MAJORSUBJECT ="工学部・工学科"
# 以前から交流会: relationship_with_individual_or_organization  きっかけに交流: active_individual_or_organization_relationship 
MOTIVE = "active_individual_or_organization_relationship" 
HOWTOKNOW = "student"

# 名字と名前を分割
lastname = NAME.split()[0]
firstname = NAME.split()[1]

#性別を変換
if GENDER == "男" or GENDER == "男性":
    gender = "male"
elif GENDER == "女" or GENDER == "女性":
    gender = "female"
elif GENDER == "その他":
    gender = "other"
elif GENDER == "無回答":
    gender = "prefer_not_to_say"

# 生年月日を分割
birth_year, birth_month, birth_day = BIRTHDAY.split('/')
print(birth_year, birth_month, birth_day )

# 入学、卒業年月を分割
encro_year, encro_month = ENCROLLMENT.split('/')
print(encro_year, encro_month)
gradu_year, gradu_month = GRADUATION.split('/')
print(gradu_year, gradu_month)


#きっかけを変換

# WebDriver のオプションを設定する
options = webdriver.ChromeOptions()
# options.add_argument('--headless')  # ヘッドレスモードを無効にする

# Chromeドライバのパスを指定する
driver = webdriver.Chrome(options=options)


def openForm(URL, TARGET):
    # Webページを開く
    driver.get(URL)
    print("ページを開きました。")
    time.sleep(2)

    target_img = driver.find_element(By.XPATH, f"//img[@alt='{TARGET}']")
    target_img.click()
    print("目標を発見してクリックした")


def writeForm():

    # 名字
    lastname_field = WebDriverWait(driver, 10).until(
        EC.presence_of_element_located((By.NAME, 'lastName'))
    )
    lastname_field.send_keys(lastname)
    print("名字を入力しました。")

    # 名前
    firstname_field = driver.find_element(By.NAME, 'firstName')
    firstname_field.send_keys(firstname)
    print("名前を入力しました。")

    # 性別
    gender_field = driver.find_element(By.XPATH, f"//input[@name='gender' and @value='{gender}']")
    gender_field.click()
    print("性別を選択しました。")

    # 年のドロップダウンを開き、特定の年を選択
    year_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[1]/div[4]/div[2]/div[1]/div[1]/div")
    year_dropdown.click()  # ドロップダウンをクリックして開く
    time.sleep(0.2)
    year_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{birth_year}']")
    year_option.click()  # 特定の年を選択
    print("年のドロップダウンを開き、特定の年を選択しました。")

    # 月のドロップダウンを開き、特定の月を選択
    month_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[1]/div[4]/div[2]/div[1]/div[2]/div")
    month_dropdown.click()  # ドロップダウンをクリックして開く
    time.sleep(0.2)
    month_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{birth_month}']")
    month_option.click()  # 特定の月を選択
    print("月のドロップダウンを開き、特定の月を選択しました。")

    # 日のドロップダウンを開き、特定の日を選択
    day_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[1]/div[4]/div[2]/div[1]/div[3]/div")
    day_dropdown.click()  # ドロップダウンをクリックして開く
    time.sleep(0.2)
    day_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{birth_day}']")
    day_option.click()  # 特定の日を選択
    print("日のドロップダウンを開き、特定の日を選択しました。")

    # メールアドレスを入力
    email_field = driver.find_element(By.NAME, 'email')
    email_field.send_keys(EMAIL)
    print("メールアドレスを入力しました。")

    # 電話番号を入力
    phone_number_field = driver.find_element(By.NAME, 'phoneNumber')
    phone_number_field.send_keys(PHONE_NUMBER)
    print("電話番号を入力しました。")

    #応援団体との関係のドロップダウンを開く
    relation_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[2]/div[1]/div[2]/div")
    relation_dropdown.click()  # ドロップダウンをクリックして開く
    time.sleep(0.2)
    relation_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{RELATION}']")
    relation_option.click()  # 特定の日を選択
    print("応援団体との関係を選択（他大学の学部生のみ対応）")
    time.sleep(0.2)

    #大学名
    uni_field = driver.find_element(By.NAME, 'universityName')
    uni_field.send_keys(MYUNIVERSITY)
    print("大学名を入力しました")

    #入学年月のドロップダウン
    # 入学年のドロップダウンを開き、特定の年を選択
    enrollment_year_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[2]/div[3]/div[2]/div[1]/div[1]")
    enrollment_year_dropdown.click()  # ドロップダウンをクリックして開く
    time.sleep(0.2)
    enrollment_year_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{encro_year}']") 
    enrollment_year_option.click()
    print("入学年を選択しました。")
    

    # 入学月のドロップダウンを開き、特定の月を選択
    enrollment_month_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[2]/div[3]/div[2]/div[1]/div[2]")
    enrollment_month_dropdown.click()
    time.sleep(0.2)
    enrollment_month_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{encro_month}']") 
    enrollment_month_option.click()
    print("入学月を選択しました。")

    # 卒業見込み年のドロップダウンを開き、特定の年を選択
    graduation_year_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[2]/div[4]/div[2]/div[1]/div[1]")
    graduation_year_dropdown.click()
    time.sleep(0.2)
    graduation_year_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{gradu_year}']")  
    graduation_year_option.click()
    print("卒業見込み年を選択しました。")

    # 卒業見込み月のドロップダウンを開き、特定の月を選択
    graduation_month_dropdown = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/section[2]/div[4]/div[2]/div[1]/div[2]")
    graduation_month_dropdown.click()
    time.sleep(0.2)
    graduation_month_option = driver.find_element(By.XPATH, f"//div[@role='option' and text()='{gradu_month}']") 
    graduation_month_option.click()
    print("卒業見込み月を選択しました。")

    # 文系・理系
    major_field = driver.find_element(By.XPATH, f"//input[@name='majorCategory' and @value='{MAJOR}']")
    major_field.click()
    print(f"{MAJOR}を選択しました。")

    #学部学科
    majorsub_field = driver.find_element(By.NAME, 'department')
    majorsub_field.send_keys(MAJORSUBJECT)
    print("学部学科を入力した")
    

    # 投票理由
    motive_field = driver.find_element(By.XPATH, f"//input[@name='reasonForVoting' and @value='{MOTIVE}']")
    motive_field.click()
    print("投票理由を選択しました。")

    #どのようにしてgivingcampainを知ったか
    howtoknow_field = driver.find_element(By.XPATH, f"//input[@name='gc2024Sources' and @value='{HOWTOKNOW}']")
    howtoknow_field.click()
    print("どのようにして知ったか選択した")



    #ポリシーok、限定情報受け取らない
    pol_field = driver.find_element(By.XPATH, "//input[@name='policyApproval']")
    pol_field.click()
    gcpor_field = driver.find_element(By.XPATH, "//input[@name='gcPortalParticipationDesired']")
    gcpor_field.click()


    time.sleep(0.2)
    # フォームを送信
    submit_button = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/div/div/form/div/button")
    submit_button.click()
    print("フォームを送信しました。")
    

def smsAuth():
    auth_code = ''
    time.sleep(1)
    print("SMSの番号を入力してください")
    while len(auth_code) != 8:
        print("8桁入力してください")
        auth_code = input()
    

    sms_field = driver.find_element(By.NAME, 'smsCode')
    sms_field.send_keys(auth_code)
    time.sleep(0.2)

    #認証コードを押す
    submit_button = driver.find_element(By.XPATH, "/html/body/div[1]/main/div[2]/form/div[2]/button")
    submit_button.click()
    time.sleep(1)


def vote(auto:bool):
    print("応援したい内容を入力します")
    if auto:
        content = "琉球大学robotサークルです！応援してます"
    else:
        content =""
    while len(content) == 0:
        if not auto:
            content = input()
    
    #入力
    message_field = driver.find_element(By.NAME, 'message')
    message_field.send_keys(content)
    time.sleep(0.2)

    #送信
    submit_button = driver.find_element(By.XPATH, "/html/body/div/main/div[2]/form/div[2]/button")
    submit_button.click()
    time.sleep(1)


def loop():
    print("終了しますか？(y/n)")
    while True:
        text = input()
        if text == 'y':
            break


if __name__ == "__main__":
    print("URL(/form/vote/step1の形になってるもの): ")
    URL = input()
    print("応援する団体名(webに書かれているもの):")
    TARGET = input()
    openForm(URL, TARGET)

    writeForm()

    
    smsAuth()

    vote(True)

    loop()
    
    # ブラウザを閉じる
    driver.quit()
    print("ブラウザを閉じました。")

