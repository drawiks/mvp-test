export namespace formula {
	
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
	export class Variable {
	    id: string;
	    name: string;
	    expression: string;
	
	    static createFrom(source: any = {}) {
	        return new Variable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.expression = source["expression"];
	    }
	}

}

export namespace main {
	
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
	    HeroDamage: number;
	    DamageTaken: number;
	    TowerDamage: number;
	    TimeDead: number;
	    StunDuration: number;
	    GoldSpentWards: number;
	    GoldSpentSmoke: number;
	    GoldSpentDust: number;
	    CampsStacked: number;
	    CreepsStacked: number;
	    RunePickups: number;
	    FirstBlood: boolean;
	    BuffsDuration: number;
	    Save: number;
	    Purge: number;
	    ShieldUptime: number;
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
	    GoldLost: number;
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
	        this.HeroDamage = source["HeroDamage"];
	        this.DamageTaken = source["DamageTaken"];
	        this.TowerDamage = source["TowerDamage"];
	        this.TimeDead = source["TimeDead"];
	        this.StunDuration = source["StunDuration"];
	        this.GoldSpentWards = source["GoldSpentWards"];
	        this.GoldSpentSmoke = source["GoldSpentSmoke"];
	        this.GoldSpentDust = source["GoldSpentDust"];
	        this.CampsStacked = source["CampsStacked"];
	        this.CreepsStacked = source["CreepsStacked"];
	        this.RunePickups = source["RunePickups"];
	        this.FirstBlood = source["FirstBlood"];
	        this.BuffsDuration = source["BuffsDuration"];
	        this.Save = source["Save"];
	        this.Purge = source["Purge"];
	        this.ShieldUptime = source["ShieldUptime"];
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
	        this.GoldLost = source["GoldLost"];
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

