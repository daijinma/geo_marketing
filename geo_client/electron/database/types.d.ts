// 临时类型定义，直到安装 @types/better-sqlite3
declare module 'better-sqlite3' {
  namespace Database {
    interface Database {
      prepare(sql: string): Statement;
      exec(sql: string): void;
      pragma(pragma: string, options?: any): any;
      close(): void;
    }

    interface Statement {
      run(...params: any[]): RunResult;
      get(...params: any[]): any;
      all(...params: any[]): any[];
    }

    interface RunResult {
      changes: number;
      lastInsertRowid: number | bigint;
    }

    interface Options {
      readonly?: boolean;
      fileMustExist?: boolean;
      timeout?: number;
      verbose?: (message?: any, ...additionalArgs: any[]) => void;
    }
  }

  interface DatabaseConstructor {
    new (filename: string, options?: Database.Options): Database.Database;
    (filename: string, options?: Database.Options): Database.Database;
  }

  const Database: DatabaseConstructor;
  export = Database;
}
