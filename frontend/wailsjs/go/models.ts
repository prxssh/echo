export namespace torrent {
    export class File {
        length: number;
        path: string[];

        static createFrom(source: any = {}) {
            return new File(source);
        }

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.length = source['length'];
            this.path = source['path'];
        }
    }
    export class Info {
        infoHash: number[];
        name: string;
        files?: File[];
        pieceLength: number;
        pieces: number[][];
        private: boolean;

        static createFrom(source: any = {}) {
            return new Info(source);
        }

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.infoHash = source['infoHash'];
            this.name = source['name'];
            this.files = this.convertValues(source['files'], File);
            this.pieceLength = source['pieceLength'];
            this.pieces = source['pieces'];
            this.private = source['private'];
        }

        convertValues(a: any, classs: any, asMap: boolean = false): any {
            if (!a) {
                return a;
            }
            if (a.slice && a.map) {
                return (a as any[]).map((elem) =>
                    this.convertValues(elem, classs)
                );
            } else if ('object' === typeof a) {
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
    export class Metainfo {
        info?: Info;
        announceUrls: string[];
        // Go type: time
        creationDate: any;
        comment: string;
        encoding: string;
        size: number;

        static createFrom(source: any = {}) {
            return new Metainfo(source);
        }

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.info = this.convertValues(source['info'], Info);
            this.announceUrls = source['announceUrls'];
            this.creationDate = this.convertValues(
                source['creationDate'],
                null
            );
            this.comment = source['comment'];
            this.encoding = source['encoding'];
            this.size = source['size'];
        }

        convertValues(a: any, classs: any, asMap: boolean = false): any {
            if (!a) {
                return a;
            }
            if (a.slice && a.map) {
                return (a as any[]).map((elem) =>
                    this.convertValues(elem, classs)
                );
            } else if ('object' === typeof a) {
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
    export class Torrent {
        metainfo?: Metainfo;
        uploaded: number;
        downloaded: number;
        left: number;

        static createFrom(source: any = {}) {
            return new Torrent(source);
        }

        constructor(source: any = {}) {
            if ('string' === typeof source) source = JSON.parse(source);
            this.metainfo = this.convertValues(source['metainfo'], Metainfo);
            this.uploaded = source['uploaded'];
            this.downloaded = source['downloaded'];
            this.left = source['left'];
        }

        convertValues(a: any, classs: any, asMap: boolean = false): any {
            if (!a) {
                return a;
            }
            if (a.slice && a.map) {
                return (a as any[]).map((elem) =>
                    this.convertValues(elem, classs)
                );
            } else if ('object' === typeof a) {
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
}
