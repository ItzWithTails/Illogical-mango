<div dir="rtl">

<h1 align="center">Illogical-mango</h1>

<p align="center">
  <b>صَدَفة سطح مكتب كاملة لـ MangoWM، مبنية على Quickshell</b>
</p>

</div>

<p align="center">
  <sub>
    <a href="../../README.md">English</a> · <a href="README.ru.md">Русский</a> · <a href="README.es.md">Español</a> · <a href="README.zh.md">中文</a> · <a href="README.ja.md">日本語</a> · <a href="README.pt.md">Português</a> · <a href="README.fr.md">Français</a> · <a href="README.de.md">Deutsch</a> · <a href="README.ko.md">한국어</a> · <a href="README.hi.md">हिन्दी</a> · <a href="README.ar.md">العربية</a> · <a href="README.it.md">Italiano</a>
  </sub>
</p>

---

<div dir="rtl">

## هذا النقل كتبه ذكاء اصطناعي. بالكامل. وهذا الملف أيضًا، بنسبة 90٪ تقريبًا

المشروع مزحة. لم يجتهد فيه أحد.

النقل إلى MangoWM - أي `services/MangoService.qml` و
`services/deferred/MangoKeybinds.qml` وإعادة كتابة اكتشاف المُركِّب في
`services/CompositorService.qml` وتغييرات المُثبِّت و doctor - كُتب من أوله إلى آخره عبر
Claude.
ليس "بمساعدته". بل هو من كتبه.

هذا مذكور في الأعلى تمامًا كي لا تكتشفه لاحقًا، من فرق في الشيفرة أو من علّة. ليس إنجازًا ولا
يُقدَّم على أنه إنجاز. في الأصل صنعت هذا النقل لنفسي، للتسلية. ضع ذلك في حسبانك.

الصَّدَفة التي تحت طبقة النقل هي iNiR من snowarch، وقد كتبها (على ما آمل) إنسان.

---

## ما هذا

بصراحة، إن سبق أن نصّبت بنفسك مُركِّب Wayland عاريًا، فلا حاجة لأن يشرح لك أحد لماذا يحتاج
إلى صَدَفة. لكنني ملزم بأن أشرح كيف يعمل.

</div>

```
تطبيقاتك
   ↓
Illogical-mango   الشريط، الرصيف، الأشرطة الجانبية، النظرة العامة، الإشعارات، الإعدادات، شاشة القفل
   ↓
Quickshell        بيئة تشغيل QML لصَدَفات Wayland
   ↓
MangoWM           النوافذ والرسم
   ↓
Wayland → GPU
```

<div dir="rtl">

**ما الذي يميّزه عن بقية إعدادات Quickshell:**

- **عائلتا لوحات كاملتان في تثبيت واحد.** Material ii (شريط عائم، أشرطة جانبية، رصيف)
  و Waffle (شريط مهام سفلي، قائمة ابدأ، مركز الإجراءات). ليستا سمتين فوق الودجات نفسها —
  بل شجرتا لوحات منفصلتان، لكل منهما نظام رموزها الخاص، وتتبدلان أثناء التشغيل بـ
  <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>.
- **تنسيق ألوان للنظام كله، لا للصَّدَفة وحدها.** خلفية واحدة تولّد لوحة ألوان Material You
  تُكتب إلى GTK3/4 و Qt وعشر طرفيات وأدوات TUI و Firefox و Discord و Spicetify و Steam
  و SDDM.
- **يُضبط دون تعديل الشيفرة.** كل شيء إعداد في الواجهة الرسومية فوق ملف `config.json` واحد.
  لن تحتاج أبدًا إلى لمس QML لتغيير المظهر أو السلوك.
- **مسار تثبيت وترقية حقيقي.** يتكفّل `./setup` بالاعتماديات وإعداد النظام؛ و`ilmango update`
  يجلب التحديثات ويشغّل ترحيلات المخطط ويحافظ على تعديلاتك ويستطيع التراجع.

**النسب.** [illogical-impulse من end-4](https://github.com/end-4/dots-hyprland) (ملفات
Hyprland) ← [iNiR من snowarch](https://github.com/snowarch/iNiR) (أُعيدت كتابته لأجل niri)
← هذا، منقولًا إلى MangoWM. لا تزال الأداة السطرية ومسارات الإعداد والبنية الداخلية تحمل اسم
`ilmango`. أما التثبيتات من عهد iNiR فينقلها الترحيل 037، وهو يترك روابط رمزية على المسارات
القديمة كي تستمر الاختصارات والسكربتات الموجودة في العمل.
ولماذا لم أشتقّ من end-4 مباشرة؟ المنطق بسيط - المشروع الذي نُقل مرة يسهل نقله مرة أخرى.
وللتشبيه، خذ Void Linux. ثبّت عليه systemd وسيعمل بلا مشاكل.
وخذ Arch Linux وانزع منه systemd، عندئذ ستضطر إلى تغيير قاعدة الحزم كلها تقريبًا.


## المُركِّب

مصنوع لأجل [MangoWM](https://github.com/DreamMaoMao/mango) ومُختبَر عليه وحده.

تتحدث الصَّدَفة إلى mango عبر مقبس IPC الخاص به على `$MANGO_INSTANCE_SIGNATURE`، وهو يرسل
لقطة كاملة للجلسة عند كل تغيير. mango على طراز dwm — وسوم، لا قائمة مساحات عمل — لذا يربط
`MangoService` أزواج `(الشاشة، رقم الوسم)` بنموذج مساحات العمل نفسه الذي يتوقعه أصلًا الشريط
والرصيف والنظرة العامة وشريط مساحات العمل، فتعمل تلك الوحدات دون تعديل.

الإعداد غير مُتلِف عن قصد. يقرأ mango ملفًا واحدًا بالضبط
(`~/.config/mango/config.conf`) ولا يدمج شيئًا، لذا لا يستبدل المُثبِّت إعدادات مُركِّبك أبدًا.
يضع اختصارات الصَّدَفة وبدء تشغيلها التلقائي في `~/.config/mango/ilmango.conf` ويلحق سطرًا واحدًا
`source-optional=` يشير إليه، دون أن يمسّ إدارة نوافذك. وبدء التشغيل التلقائي سطر
`exec-once=ilmango run --daemon` في ذلك الملف، لا وحدة systemd.

> [!NOTE]
> **شيفرة niri و Hyprland ما زالت في الشجرة.** بقيت `NiriService.qml` و`HyprlandData.qml`
> والفروع `isNiri` / `isHyprland` من المنبع وما زالت تُترجَم. هي موروثة لا مدعومة: لا شيء
> هنا مُختبَر على تلك المُركِّبات ولا شيء مصان لأجلها. إن أردت niri فخذ
> [iNiR الأصلي](https://github.com/snowarch/iNiR).

---

## لقطات الشاشة

عائلتا اللوحات كلتاهما، منقولتان من المنبع دون تغيير.

</div>

<details open>
<summary><b>Material ii</b>: شريط عائم، أشرطة جانبية، جماليات Material Design</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/1fe258bc-8aec-4fd9-8574-d9d7472c3cc8) | ![](https://github.com/user-attachments/assets/3ce2055b-648c-45a1-9d09-705c1b4a03b7) |
| ![](https://github.com/user-attachments/assets/ea2311dc-769e-44dc-a46d-37cf8807d2cc) | ![](https://github.com/user-attachments/assets/ba866063-b26a-47cb-83c8-d77bd033bf8b) |
| ![](https://github.com/user-attachments/assets/88e76566-061b-4f8c-a9a8-53c157950138) | |

</details>

<details>
<summary><b>Waffle</b>: شريط مهام سفلي، مركز إجراءات، أجواء Windows 11</summary>

| | |
|:---:|:---:|
| ![](https://github.com/user-attachments/assets/5c5996e7-90eb-4789-9921-0d5fe5283fa3) | ![](https://github.com/user-attachments/assets/fadf9562-751e-4138-a3a1-b87b31114d44) |

</details>

---

<div dir="rtl">

> [!WARNING]
> الإعداد الافتراضي موجَّه إلى عتاد حديث نسبيًا. على الأجهزة الضعيفة أطفئ المؤثرات، واحذف
> اللوحات التي لا تحتاجها، وسطِّح النمط البصري — كل ذلك يتم من الإعدادات أو عبر
> `config.json`.

## المزايا

**عائلتا لوحات**، تتبدلان أثناء التشغيل بـ <kbd>Super</kbd>+<kbd>Shift</kbd>+<kbd>W</kbd>:

- **Material ii** — شريط عائم، أشرطة جانبية، رصيف، و8 أنماط بصرية (Material و Cards
  و Aurora و Illogical-mango و Angel و Regalia و ZZZ و Cookie Shapes)
- **Waffle** — شريط مهام على طراز Windows 11، قائمة ابدأ، مركز إجراءات، مركز إشعارات

**تنسيق ألوان تلقائي.** اختر خلفية ويتبعها النظام كله: تنتشر ألوان Material You الخاصة
بالصَّدَفة إلى GTK3/4 و Qt والطرفيات و Firefox و Discord و Spicetify و Steam و SDDM. تأتي معه
إعدادات Regalia و Gruvbox و Catppuccin و Rosé Pine الجاهزة، أو اصنع واحدة لك.

<details>
<summary><b>قائمة المزايا الكاملة</b></summary>

### التنسيق والمظهر

- **8 أنماط بصرية**: Material (صُلب)، Cards، Aurora (ضبابية زجاجية)، Illogical-mango (مستوحى من TUI)، Angel (وحشية جديدة)، Regalia (هيكل هندسي أسود، حبر عاجي دافئ، معدن شمبانيا متحفظ)، ZZZ (ألواح ملصقات)، Cookie Shapes (تحوّل متحرك للأشكال)
- **ألوان ديناميكية من الخلفية** عبر Material You، تنتشر في النظام كله
- **10 طرفيات وأدوات TUI تُنسَّق تلقائيًا**: foot و kitty و alacritty و ghostty و wezterm و starship و fuzzel و btop و lazygit و yazi
- **تنسيق التطبيقات**: GTK3/4، Qt (عبر plasma-integration و darkly)، Firefox (MaterialFox)، Discord/Vesktop (System24)، Zed، Spicetify، Steam، SDDM
- **إعدادات سمات جاهزة**: Regalia و Regalia Ivory و Gruvbox و Catppuccin و Rosé Pine ومخصصة
- **خلفيات فيديو**: mp4/webm/gif مع ضبابية اختيارية، أو إطار أول مجمَّد من أجل الأداء
- **ودجات سطح المكتب**: ساعة (أنماط عدة)، طقس، تحكم بالوسائط على طبقة الخلفية

### الشريط

- **6 أنماط للشريط**: classic و islands و scenic و frame وكبسولات Material 3 و pill
- **شريط pill**: جزيرة وسطى تتحوّل وتنفتح عند المرور فوقها إلى مساحات العمل والمُشغِّل والخالط والوسائط والتقويم ومسجّل الشاشة
- **تخطيط معياري** مع محرر سحب في الإعدادات، فأي وحدة تذهب إلى أي مكان
- **شريط عمودي** لمن يريد حافة الشاشة

### الأشرطة الجانبية والودجات (Material ii)

الشريط الأيسر (درج التطبيقات):
- **AI Chat**: فهارس نماذج حيّة عبر Ollama و LM Studio و OpenRouter و Gemini و Groq و Mistral و Cerebras و Anthropic و OpenAI و OpenCode
- **YT Music**: مشغّل InnerTube بلا كعكات، مع بحث وقائمة انتظار وراديو وكلمات متزامنة
- **متصفح Wallhaven**: ابحث عن الخلفيات وطبّقها مباشرة
- **متتبّع الأنمي**: تكامل مع AniList مع عرض جدول العرض
- **مترجم**: عبر Gemini أو translate-shell
- **ودجات قابلة للسحب**: العملات الرقمية، مشغّل الوسائط، ملاحظات سريعة، حلقات الحالة، تقويم أسبوعي

الشريط الأيمن:
- **تقويم** مع تكامل الأحداث
- **مركز الإشعارات**
- **مفاتيح سريعة**: WiFi، بلوتوث، الإضاءة الليلية، عدم الإزعاج، أنماط الطاقة، WARP VPN، EasyEffects
- **خالط الصوت** مع تحكم لكل تطبيق
- **إدارة أجهزة** البلوتوث و WiFi
- **مؤقت بومودورو**، **قائمة مهام**، **آلة حاسبة**، **مفكرة**
- **مراقب النظام**: المعالج والذاكرة والحرارة

### الأدوات

- **نظرة عامة على مساحات العمل**: بحث عن التطبيقات وآلة حاسبة، موضوعة فوق نموذج وسوم mango
- **لوحة معلومات**: تراكب من ثلاثة أعمدة قابل للضبط، فيه الأجندة والإشعارات والمهام والملاحظات والوسائط والطقس
- **شريط مساحات العمل عند الحافة**: قضيب يظهر عند المرور، مع معاينات حيّة وإعادة ترتيب بالسحب
- **مبدّل النوافذ**: Alt-Tab متحرك عبر كل مساحات العمل، اختياري
- **مدير الحافظة**: سجل مع بحث ومعاينة للصور
- **أدوات المنطقة**: لقطات شاشة، تسجيل شاشة، OCR، بحث عكسي بالصور
- **ورقة الاختصارات**: عارض للاختصارات مأخوذ من إعدادات mango عندك
- **التحكم بالوسائط**: مشغّل MPRIS كامل مع عدة تخطيطات جاهزة
- **العرض على الشاشة**: مؤشرات الصوت والسطوع والوسائط
- **التعرّف على الأغاني**: على طريقة Shazam، عبر SongRec
- **الإدخال الصوتي**: whisper.cpp محليًا إن كان مثبتًا، أو خلفية موصولة من Groq أو Gemini أو OpenAI

### النظام

- **إعدادات رسومية**: اضبط كل شيء دون لمس أي ملف
- **GameMode**: يعطّل المؤثرات تلقائيًا للتطبيقات في وضع ملء الشاشة
- **تحديثات تلقائية**: `ilmango update` مع تراجع وترحيلات وحفظ لتعديلاتك
- **شاشة قفل** و**شاشة جلسة** (خروج/إعادة تشغيل/إيقاف/تعليق)
- **وكيل polkit**، **لوحة مفاتيح على الشاشة**، **مدير بدء تشغيل تلقائي** مبني على سطر `exec-once` في إعدادات mango
- **Kira**: تميمة اختيارية برسوم البكسل، تتجول عند حواف الشاشة وتتفاعل مع ما تفعله. مطفأة افتراضيًا، وحزمة الرسوم بحجم ~32 ميبي بايت تُنزَّل على حدة من `./setup` › Extras
- **15 لغة** مع كشف تلقائي
- **إضاءة ليلية**: بجدول أو يدويًا
- **الطقس**: Open-Meteo، يدعم GPS أو إحداثيات يدوية أو اسم المدينة
- **إدارة البطارية**: عتبات قابلة للضبط، تعليق تلقائي عند المستوى الحرج
- **أصوات أحداث مخصصة** مع مستوى صوت عام وملف صوتي لكل حدث

</details>

---

## بداية سريعة (المُثبِّت سيكون غيره مستقبلًا)

</div>

```bash
git clone https://github.com/ItzWithTails/illogical-mango.git
cd Illogical-mango
./setup install       # تفاعلي، يسأل قبل كل خطوة
./setup install -y    # تلقائي، بلا أسئلة
```

<div dir="rtl">

يتكفّل المُثبِّت بالاعتماديات وإعداد النظام وتنسيق الألوان. يكتب اختصارات الصَّدَفة في
`~/.config/mango/ilmango.conf` ويوصلها بإعدادات mango الموجودة عندك دون أن يمسّ إدارة نوافذك.
أعد تشغيل mango أو نفّذ `mmsg dispatch reload_config`.

</div>

```bash
ilmango run                        # تشغيل الصَّدَفة
ilmango settings                   # فتح واجهة الإعدادات
ilmango logs                       # الاطلاع على السجلات
ilmango doctor                     # تشخيص وإصلاح تلقائي
ilmango update                     # جلب + ترحيلات + إعادة تشغيل
```

<div dir="rtl">

مداخل أخرى:

</div>

```bash
./setup                 # قائمة TUI، اختر ما تريد
./setup install --skip-mango    # لا تمسّ إعدادات mango إطلاقًا
sudo make install       # تثبيت على مستوى النظام بدل مجلد المنزل
./setup rollback        # التراجع عن آخر تحديث
```

<div dir="rtl">

**التوزيعات.** Arch هو الهدف الأساسي والأكثر اختبارًا. أما Debian و Fedora فالنقل موجود
لهما بالطبع… على مسؤوليتك، لم يجرِ عليهما أي اختبار.

---

## الاختصارات

تُثبَّت من `defaults/mango/config.conf`:

| المفتاح | الإجراء |
|-----|--------|
| <kbd>Super</kbd> + <kbd>Space</kbd> | نظرة عامة: بحث عن التطبيقات، تنقّل بين الوسوم |
| <kbd>Super</kbd> + <kbd>V</kbd> | سجل الحافظة |
| <kbd>Super</kbd> + <kbd>P</kbd> | الشريط الجانبي الأيسر |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>N</kbd> | الشريط الجانبي الأيمن |
| <kbd>Super</kbd> + <kbd>D</kbd> | لوحة المعلومات |
| <kbd>Super</kbd> + <kbd>,</kbd> | الإعدادات |
| <kbd>Super</kbd> + <kbd>/</kbd> | ورقة الاختصارات |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>W</kbd> | تبديل عائلة اللوحات |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>S</kbd> | لقطة لمنطقة |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>X</kbd> | OCR لمنطقة |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>R</kbd> | تسجيل منطقة |
| <kbd>Super</kbd> + <kbd>Shift</kbd> + <kbd>C</kbd> | بحث عكسي عن منطقة |
| <kbd>Super</kbd> + <kbd>L</kbd> | قفل |
| <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>Delete</kbd> | شاشة الجلسة |

اختصارات إدارة النوافذ ملكك — الصَّدَفة لا تعرّفها. المرجع الكامل:
[الاختصارات](../KEYBINDS.md).

---

## الخلفيات

تأتي معه 15 خلفية. وللمزيد انظر [iNiR-Walls](https://github.com/snowarch/iNiR-Walls)،
وهي مجموعة تنسجم جيدًا مع مسار Material You.

---

## التوثيق (لأجل niri، لا لأجل mango)

| الصفحة | ما فيها |
|---|---|
| [التثبيت](../INSTALL.md) | كيف تجعله يعمل |
| [Setup](../SETUP.md) | التحديثات والترحيلات والتراجع |
| [الاختصارات](../KEYBINDS.md) | كل التركيبات |
| [IPC](../IPC.md) | أهداف يمكن ربطها بمفتاح أو استدعاؤها من سكربت |
| [الحزم](../PACKAGES.md) | كل اعتمادية ولماذا هي موجودة |
| [القيود](../LIMITATIONS.md) | ما يُعرف أنه معطوب، وطرق الالتفاف |
| [المُركِّبات](../COMPOSITORS.md) | كيف يعمل التكامل مع المُركِّب |
| [البنية](../../ARCHITECTURE.md) | كيف رُكِّبت الشيفرة |

معظم `docs/` موروث من المنبع ولا يزال يصف niri في مواضع. وحيثما اختلف التوثيق مع هذا الملف
حول المُركِّب المدعوم، فالصواب في هذا الملف.

---

## حل المشكلات

</div>

```bash
ilmango logs                       # سجلات التشغيل الأخيرة
ilmango restart                    # إعادة تشغيل بيئة التشغيل الحالية
ilmango repair                     # doctor + إعادة تشغيل + فحص مُرشَّح للسجلات
ilmango doctor                     # تشخيص وإصلاح تلقائي للمشكلات الشائعة
./setup rollback                # التراجع عن آخر تحديث
claude "ساعدني من فضلك"          # إن لم ترغب في البحث بنفسك. هيا، عليه أن يستحق العشرين دولارًا
```

<div dir="rtl">

ألقِ نظرة على [القيود](../LIMITATIONS.md) لتضحك قليلًا.

---

## المساهمة

انظر [CONTRIBUTING.md](../../CONTRIBUTING.md) — تهيئة بيئة التطوير، وأنماط الشيفرة، وقواعد
طلبات السحب.


---

## شكر وتقدير

- [**snowarch**](https://github.com/snowarch/iNiR): iNiR، الصَّدَفة المنقولة هنا
- [**end-4**](https://github.com/end-4/dots-hyprland): illogical-impulse، الذي اشتُقّ منه iNiR
- [**Gakuseei**](https://github.com/Gakuseei): [Ricelin](https://github.com/Gakuseei/Ricelin)، ومنه جاء شريط pill ومظهرا washi و flame
- [**Quickshell**](https://quickshell.outfoxxed.me/): إطار العمل الذي يعمل عليه هذا
- [**MangoWM**](https://github.com/DreamMaoMao/mango): المُركِّب الذي صُنع لأجله
- **Claude** (Anthropic): كتب النقل إلى MangoWM، كما ذُكر في الأعلى تمامًا

رخصة GPL-3.0، مثل ملفات end-4. حقوق النشر للمنبع (C) 2025-2026 snowarch.

</div>
