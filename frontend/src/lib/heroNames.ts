/**
 * Hero display names keyed by the Dota 2 API name (the field odota/parser
 * emits in "players[].hero"). Sourced from opendota /api/heroes
 * (localized_name), 127 heroes.
 */

const HERO_NAMES: Record<string, string> = {
  "antimage": "Anti-Mage",
  "axe": "Axe",
  "bane": "Bane",
  "bloodseeker": "Bloodseeker",
  "crystal_maiden": "Crystal Maiden",
  "drow_ranger": "Drow Ranger",
  "earthshaker": "Earthshaker",
  "juggernaut": "Juggernaut",
  "mirana": "Mirana",
  "morphling": "Morphling",
  "nevermore": "Shadow Fiend",
  "phantom_lancer": "Phantom Lancer",
  "puck": "Puck",
  "pudge": "Pudge",
  "razor": "Razor",
  "sand_king": "Sand King",
  "storm_spirit": "Storm Spirit",
  "sven": "Sven",
  "tiny": "Tiny",
  "vengefulspirit": "Vengeful Spirit",
  "windrunner": "Windranger",
  "zuus": "Zeus",
  "kunkka": "Kunkka",
  "lina": "Lina",
  "lion": "Lion",
  "shadow_shaman": "Shadow Shaman",
  "slardar": "Slardar",
  "tidehunter": "Tidehunter",
  "witch_doctor": "Witch Doctor",
  "lich": "Lich",
  "riki": "Riki",
  "enigma": "Enigma",
  "tinker": "Tinker",
  "sniper": "Sniper",
  "necrolyte": "Necrophos",
  "warlock": "Warlock",
  "beastmaster": "Beastmaster",
  "queenofpain": "Queen of Pain",
  "venomancer": "Venomancer",
  "faceless_void": "Faceless Void",
  "skeleton_king": "Wraith King",
  "death_prophet": "Death Prophet",
  "phantom_assassin": "Phantom Assassin",
  "pugna": "Pugna",
  "templar_assassin": "Templar Assassin",
  "viper": "Viper",
  "luna": "Luna",
  "dragon_knight": "Dragon Knight",
  "dazzle": "Dazzle",
  "rattletrap": "Clockwerk",
  "leshrac": "Leshrac",
  "furion": "Nature's Prophet",
  "life_stealer": "Lifestealer",
  "dark_seer": "Dark Seer",
  "clinkz": "Clinkz",
  "omniknight": "Omniknight",
  "enchantress": "Enchantress",
  "huskar": "Huskar",
  "night_stalker": "Night Stalker",
  "broodmother": "Broodmother",
  "bounty_hunter": "Bounty Hunter",
  "weaver": "Weaver",
  "jakiro": "Jakiro",
  "batrider": "Batrider",
  "chen": "Chen",
  "spectre": "Spectre",
  "ancient_apparition": "Ancient Apparition",
  "doom_bringer": "Doom",
  "ursa": "Ursa",
  "spirit_breaker": "Spirit Breaker",
  "gyrocopter": "Gyrocopter",
  "alchemist": "Alchemist",
  "invoker": "Invoker",
  "silencer": "Silencer",
  "obsidian_destroyer": "Outworld Destroyer",
  "lycan": "Lycan",
  "brewmaster": "Brewmaster",
  "shadow_demon": "Shadow Demon",
  "lone_druid": "Lone Druid",
  "chaos_knight": "Chaos Knight",
  "meepo": "Meepo",
  "treant": "Treant Protector",
  "ogre_magi": "Ogre Magi",
  "undying": "Undying",
  "rubick": "Rubick",
  "disruptor": "Disruptor",
  "nyx_assassin": "Nyx Assassin",
  "naga_siren": "Naga Siren",
  "keeper_of_the_light": "Keeper of the Light",
  "wisp": "Io",
  "visage": "Visage",
  "slark": "Slark",
  "medusa": "Medusa",
  "troll_warlord": "Troll Warlord",
  "centaur": "Centaur Warrunner",
  "magnataur": "Magnus",
  "shredder": "Timbersaw",
  "bristleback": "Bristleback",
  "tusk": "Tusk",
  "skywrath_mage": "Skywrath Mage",
  "abaddon": "Abaddon",
  "elder_titan": "Elder Titan",
  "legion_commander": "Legion Commander",
  "techies": "Techies",
  "ember_spirit": "Ember Spirit",
  "earth_spirit": "Earth Spirit",
  "abyssal_underlord": "Underlord",
  "terrorblade": "Terrorblade",
  "phoenix": "Phoenix",
  "oracle": "Oracle",
  "winter_wyvern": "Winter Wyvern",
  "arc_warden": "Arc Warden",
  "monkey_king": "Monkey King",
  "dark_willow": "Dark Willow",
  "pangolier": "Pangolier",
  "grimstroke": "Grimstroke",
  "hoodwink": "Hoodwink",
  "void_spirit": "Void Spirit",
  "snapfire": "Snapfire",
  "mars": "Mars",
  "ringmaster": "Ringmaster",
  "dawnbreaker": "Dawnbreaker",
  "marci": "Marci",
  "primal_beast": "Primal Beast",
  "muerta": "Muerta",
  "kez": "Kez",
  "largo": "Largo",
};

/** Lowercase, no spaces/symbols: "Queen of Pain" and "queenofpain" agree. */
export function normalizeHeroName(hero: string): string {
  const key = hero
    .trim()
    .toLowerCase()
    .replace(/^npc_dota_hero_/, "")
    .replace(/[^a-z0-9_]/g, "");
  // The odota parser derives the "hero" string from the in-game unit name
  // (snake_case of CDOTA_Unit_Hero_*), which differs from the Valve/opendota
  // api slug for a few legacy heroes. Normalize those so CDN icons and this
  // map line up. (Backend also canonicalizes by hero_id; this is the fallback.)
  if (HERO_ALIASES[key]) return HERO_ALIASES[key];
  return key;
}

/** Unit-derived snake_case -> valve/opendota api slug, for the legacy mismatches. */
const HERO_ALIASES: Record<string, string> = {
  vengeful_spirit: "vengefulspirit",
  queen_of_pain: "queenofpain",
  doom: "doom_bringer",
};

/** Human-readable hero name, falling back to the raw string for unknowns. */
export function heroName(hero: string): string {
  return HERO_NAMES[normalizeHeroName(hero)] ?? hero;
}