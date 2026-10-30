"""One-off: build seeds/global.json from seeds/lagos.json.

Keeps the 8 Lagos-anchored users (incl. the demo logins) and moves 19 users
to Nairobi, London, and New York with local areas, currencies, budgets,
phones, coords, and bio touch-ups. Answers, modes, genders, and durations
are untouched.

Run once from the repo root, verify via seed import, then delete this file
and seeds/lagos.json.
"""
import json

# email -> (city, area, currency, budget_min, budget_max).
# idx = 1-based file order; phones and coords derive from it.
MOVED = {
    # Nairobi (KES)
    "funmi@example.com": ("Nairobi", "Kilimani", "KES", 30000, 45000),
    "ibrahim@example.com": ("Nairobi", "Westlands", "KES", 50000, 70000),
    "aisha@example.com": ("Nairobi", "Kileleshwa", "KES", 60000, 90000),
    "blessing@example.com": ("Nairobi", "Lavington", "KES", 25000, 40000),
    "grace@example.com": ("Nairobi", "South B", "KES", 40000, 60000),
    "seun@example.com": ("Nairobi", "Kasarani", "KES", 50000, 75000),
    "racheal@example.com": ("Nairobi", "Ruaka", "KES", 20000, 35000),
    # London (GBP)
    "tobi@example.com": ("London", "Brixton", "GBP", 650, 850),
    "sandra@example.com": ("London", "Camden", "GBP", 1500, 2000),
    "daniel@example.com": ("London", "Stratford", "GBP", 1400, 1900),
    "yusuf@example.com": ("London", "Peckham", "GBP", 700, 950),
    "peter@example.com": ("London", "Hackney", "GBP", 800, 1100),
    "chidi@example.com": ("London", "Croydon", "GBP", 550, 750),
    # New York (USD)
    "amara@example.com": ("New York", "Bushwick", "USD", 950, 1250),
    "nnamdi@example.com": ("New York", "Astoria", "USD", 1100, 1450),
    "femi@example.com": ("New York", "Harlem", "USD", 1000, 1350),
    "kemi@example.com": ("New York", "Bed-Stuy", "USD", 1100, 1450),
    "victor@example.com": ("New York", "Red Hook", "USD", 900, 1200),
    "zainab@example.com": ("New York", "Port Morris", "USD", 900, 1200),
}

LAGOS_KEEP = {
    "adaeze@example.com", "tunde@example.com", "chiamaka@example.com",
    "emeka@example.com", "ngozi@example.com", "olumide@example.com",
    "kunle@example.com", "halima@example.com",
}

# email -> (old fragment that must be present, full replacement bio)
BIO_PATCHES = {
    "aisha@example.com": (
        "around Ajah",
        "Night owl designer looking to co-search around Kileleshwa.",
    ),
    "daniel@example.com": (
        "within Ikoyi",
        "Consultant relocating within Stratford. Fine with cats.",
    ),
    "racheal@example.com": (
        "around Yaba",
        "Student hunting with a future flatmate around Ruaka.",
    ),
}

CITY_BASE = {
    "Nairobi": (-1.2921, 36.8219),
    "London": (51.5074, -0.1278),
    "New York": (40.7128, -74.0060),
}


def phone_for(city, idx):
    if city == "Nairobi":
        return f"+254-722-000-00{idx:02d}"
    if city == "London":
        return f"+44-7700-900-0{idx:02d}"
    return f"+1-212-555-01{idx:02d}"


with open("seeds/lagos.json") as f:
    users = json.load(f)

assert len(users) == 27, f"{len(users)} users"
assert set(MOVED) | LAGOS_KEEP == {u["email"] for u in users}, "assignment mismatch"

counts = {}
for i, u in enumerate(users, start=1):
    email = u["email"]
    p = u["profile"]
    assert len(u["answers"]) == 12, f"{email}: {len(u['answers'])} answers"
    for key in ("occupation", "gender", "gender_pref", "duration_months"):
        assert key in p, f"{email}: missing {key}"
    # Source file must still be the all-Lagos set (catches re-runs/edits).
    assert p["currency"] == "NGN", f"{email}: currency {p['currency']}"
    assert "Lagos" in p["area_text"], f"{email}: area {p['area_text']}"
    if email in LAGOS_KEEP:
        counts["Lagos"] = counts.get("Lagos", 0) + 1
        continue
    city, area, currency, bmin, bmax = MOVED[email]
    assert bmin < bmax and bmin > 0
    assert len(currency) <= 8
    lat0, lng0 = CITY_BASE[city]
    p["area_text"] = f"{area}, {city}"
    p["lat"] = round(lat0 + i * 0.007, 4)
    p["lng"] = round(lng0 + i * 0.009, 4)
    p["budget_min"] = bmin
    p["budget_max"] = bmax
    p["currency"] = currency
    p["contact_handle"] = phone_for(city, i)
    if email in BIO_PATCHES:
        old, new = BIO_PATCHES[email]
        assert old in p["bio"], f"{email}: fragment {old!r} not in bio"
        p["bio"] = new
    counts[city] = counts.get(city, 0) + 1

with open("seeds/global.json", "w") as f:
    json.dump(users, f, indent=2)
    f.write("\n")
print(f"wrote seeds/global.json: {counts}")
