create table user_settings(
    id bigint primary key references user_account(id) on delete cascade,
    smileys varchar(4)[] not null default '{"😀", "😂", "❤️", "❤️‍🔥", "😎", "👀", "💩", "💔", "🍒", "🍎", "🔥", "💧", "❄️", "🌎", "👍", "👎", "💣",  "⚠️", "⛔", "☢️", "☣️", "♻️", "✅", "❌", "⚡", "🚀", "#️⃣", "*️⃣", "0️⃣", "1️⃣", "2️⃣", "3️⃣", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟", "©", "™", "®"}'
);

insert into user_settings(id) select id from user_account;
