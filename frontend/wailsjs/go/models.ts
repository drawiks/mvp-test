export namespace formula {

	export class BreakdownRow {
	    label: string;
	    value: number;

	    static createFrom(source: any = {}) {
	        return new BreakdownRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.value = source["value"];
	    }
	}
	export class Preset {
	    id: string;
	    name: string;
	    kind: string;
	    weights: Record<string, number>;
	    expression: string;

	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.weights = source["weights"];
	        this.expression = source["expression"];
	    }
	}

}

export namespace main {

	export class AssistantChatMessage {
	    role: string;
	    text: string;

	    static createFrom(source: any = {}) {
	        return new AssistantChatMessage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.text = source["text"];
	    }
	}
	export class AssistantHistoryState {
	    messages: AssistantChatMessage[];
	    proposal?: AssistantProposal;

	    static createFrom(source: any = {}) {
	        return new AssistantHistoryState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], AssistantChatMessage);
	        this.proposal = this.convertValues(source["proposal"], AssistantProposal);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FormulaDiff {
	    current: string;
	    proposed: string;
	    added: string[];
	    removed: string[];

	    static createFrom(source: any = {}) {
	        return new FormulaDiff(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.proposed = source["proposed"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	    }
	}
	export class GoogleKeyInfo {
	    index: number;
	    masked: string;
	    active: boolean;
	    status: string;
	    cooldownUntil: string;
	    lastError: string;

	    static createFrom(source: any = {}) {
	        return new GoogleKeyInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.masked = source["masked"];
	        this.active = source["active"];
	        this.status = source["status"];
	        this.cooldownUntil = source["cooldownUntil"];
	        this.lastError = source["lastError"];
	    }
	}
	export class AssistantModel {
	    id: string;
	    name: string;
	    contextLength: number;

	    static createFrom(source: any = {}) {
	        return new AssistantModel(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.contextLength = source["contextLength"];
	    }
	}
	export class AssistantProposal {
	    name: string;
	    description: string;
	    expression: string;
	    actionId?: string;
	    pending?: boolean;
	    mode?: string;
	    presetId?: string;
	    targetName?: string;
	    currentExpression?: string;
	    activate?: boolean;
	    baseActiveId?: string;

	    static createFrom(source: any = {}) {
	        return new AssistantProposal(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.expression = source["expression"];
	        this.actionId = source["actionId"];
	        this.pending = source["pending"];
	        this.mode = source["mode"];
	        this.presetId = source["presetId"];
	        this.targetName = source["targetName"];
	        this.currentExpression = source["currentExpression"];
	        this.activate = source["activate"];
	        this.baseActiveId = source["baseActiveId"];
	    }
	}
	export class AssistantSettings {
	    model: string;
	    temperature: number;
	    maxToolRounds: number;
	    sendPlayerNames: boolean;
	    googleKeyCount: number;
	    googleActiveKey: number;

	    static createFrom(source: any = {}) {
	        return new AssistantSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	        this.maxToolRounds = source["maxToolRounds"];
	        this.sendPlayerNames = source["sendPlayerNames"];
	        this.googleKeyCount = source["googleKeyCount"];
	        this.googleActiveKey = source["googleActiveKey"];
	    }
	}
	export class UpdateInfo {
	    currentVersion: string;
	    latestVersion: string;
	    available: boolean;
	    notes: string;

	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.available = source["available"];
	        this.notes = source["notes"];
	    }
	}

	export class PlayerView {
	    SteamID: number;
	    PlayerID: number;
	    HeroID: number;
	    Hero: string;
	    Team: string;
	    Name: string;
	    Position: number;
	    Lane: string;
	    Kills: number;
	    Deaths: number;
	    Assists: number;
	    Level: number;
	    LastHits: number;
	    Networth: number;
	    GPM: number;
	    XPM: number;
	    Healing: number;
	    ManaRestored: number;
	    HeroDamage: number;
	    DamageTaken: number;
	    TowerDamage: number;
	    TimeDead: number;
	    StunDuration: number;
	    GoldSpentWards: number;
	    GoldSpentSmoke: number;
	    GoldSpentDust: number;
	    GoldLost: number;
	    CampsStacked: number;
	    CreepsStacked: number;
	    RunePickups: number;
	    FirstBlood: boolean;
	    BuffsDuration: number;
	    Save: number;
	    Purge: number;
	    ShieldUptime: number;
	    BuffStatsDuration: number;
	    InvisibilityDuration: number;
	    BuffHasteDuration: number;
	    FearDuration: number;
	    RootsDuration: number;
	    LeashDuration: number;
	    TrapDuration: number;
	    TauntDuration: number;
	    SilenceDuration: number;
	    BreakDuration: number;
	    DisarmDuration: number;
	    HealDuration: number;
	    HealValue: number;
	    WisdomsCaptured: number;
	    WatchersCaptured: number;
	    LotusesGathered: number;
	    CourierKills: number;
	    Score: number;
	    TeamPlace: number;
	    GlobalPlace: number;
	    IsWinner: boolean;
	    MvpRole: string;
	    Stats: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new PlayerView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SteamID = source["SteamID"];
	        this.PlayerID = source["PlayerID"];
	        this.HeroID = source["HeroID"];
	        this.Hero = source["Hero"];
	        this.Team = source["Team"];
	        this.Name = source["Name"];
	        this.Position = source["Position"];
	        this.Lane = source["Lane"];
	        this.Kills = source["Kills"];
	        this.Deaths = source["Deaths"];
	        this.Assists = source["Assists"];
	        this.Level = source["Level"];
	        this.LastHits = source["LastHits"];
	        this.Networth = source["Networth"];
	        this.GPM = source["GPM"];
	        this.XPM = source["XPM"];
	        this.Healing = source["Healing"];
	        this.ManaRestored = source["ManaRestored"];
	        this.HeroDamage = source["HeroDamage"];
	        this.DamageTaken = source["DamageTaken"];
	        this.TowerDamage = source["TowerDamage"];
	        this.TimeDead = source["TimeDead"];
	        this.StunDuration = source["StunDuration"];
	        this.GoldSpentWards = source["GoldSpentWards"];
	        this.GoldSpentSmoke = source["GoldSpentSmoke"];
	        this.GoldSpentDust = source["GoldSpentDust"];
	        this.GoldLost = source["GoldLost"];
	        this.CampsStacked = source["CampsStacked"];
	        this.CreepsStacked = source["CreepsStacked"];
	        this.RunePickups = source["RunePickups"];
	        this.FirstBlood = source["FirstBlood"];
	        this.BuffsDuration = source["BuffsDuration"];
	        this.Save = source["Save"];
	        this.Purge = source["Purge"];
	        this.ShieldUptime = source["ShieldUptime"];
	        this.BuffStatsDuration = source["BuffStatsDuration"];
	        this.InvisibilityDuration = source["InvisibilityDuration"];
	        this.BuffHasteDuration = source["BuffHasteDuration"];
	        this.FearDuration = source["FearDuration"];
	        this.RootsDuration = source["RootsDuration"];
	        this.LeashDuration = source["LeashDuration"];
	        this.TrapDuration = source["TrapDuration"];
	        this.TauntDuration = source["TauntDuration"];
	        this.SilenceDuration = source["SilenceDuration"];
	        this.BreakDuration = source["BreakDuration"];
	        this.DisarmDuration = source["DisarmDuration"];
	        this.HealDuration = source["HealDuration"];
	        this.HealValue = source["HealValue"];
	        this.WisdomsCaptured = source["WisdomsCaptured"];
	        this.WatchersCaptured = source["WatchersCaptured"];
	        this.LotusesGathered = source["LotusesGathered"];
	        this.CourierKills = source["CourierKills"];
	        this.Score = source["Score"];
	        this.TeamPlace = source["TeamPlace"];
	        this.GlobalPlace = source["GlobalPlace"];
	        this.IsWinner = source["IsWinner"];
	        this.MvpRole = source["MvpRole"];
	        this.Stats = source["Stats"];
	    }
	}

}

