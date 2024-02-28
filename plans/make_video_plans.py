import json
import sys
from enum import Enum
from typing import List, Optional, Union

import typer
import yaml
from pydantic import BaseModel
from typing_extensions import Annotated


class ActionEnum(str, Enum):
    KafkaProducer = 'KafkaProducer'
    KafkaConsumer = 'KafkaConsumer'
    RabbitMQ = 'RabbitMQ'
    RedisCommand = 'RedisCommand'
    HTTPRequest = 'HTTPRequest'
    PostgresqlQuery = 'PostgresqlQuery'
    Burn = 'Burn'
    AllocateMemory = 'AllocateMemory'
    BackgroundTask = 'BackgroundTask'
    Script = 'Script'
    UpdateBackgroundService = 'UpdateBackgroundService'


class RabbitMQProduceMessage(BaseModel):
    queue: str
    message_size: int
    message_count: int


class RabbitMQConsumeMessage(BaseModel):
    queue: str
    message_count: int


class RabbitMQConfig(BaseModel):
    url: str
    produce_messages: Optional[List[RabbitMQProduceMessage]]
    consume_messages: Optional[List[RabbitMQConsumeMessage]]


class KafkaConsumerConfig(BaseModel):
    brokers: List[str]
    username: str
    password: str
    tls_enable: bool
    sasl_enable: bool
    topic: str
    message_count: int
    message_handler: str


class KafkaProducerConfig(BaseModel):
    brokers: List[str]
    username: str
    password: str
    tls_enable: bool
    sasl_enable: bool
    topic: str
    message_size: int
    message_count: int


class RedisCommandConfig(BaseModel):
    command: str
    address: str
    args: List[str]


class HTTPRequestConfig(BaseModel):
    url: str
    body: 'Workload'


class BurnConfig(BaseModel):
    duration: str


class PostgresqlQueryConfig(BaseModel):
    host: str
    port: int
    dbname: str
    user: str
    password: str
    query: str
    repeat: int
    maxopen: int
    maxidle: int


class BackgroundTaskConfig(BaseModel):
    id: str
    duration: str
    workload: 'Workload'


class AllocateMemoryConfig(BaseModel):
    size_bytes: int
    num_allocations: int


class UpdateBackgroundServiceConfig(BaseModel):
    script: str
    name: str


class ScriptConfig(BaseModel):
    script: str


class Action(BaseModel):
    name: str
    config: Union[KafkaConsumerConfig, KafkaProducerConfig, RabbitMQConfig,
                  RedisCommandConfig, HTTPRequestConfig, PostgresqlQueryConfig,
                  AllocateMemoryConfig, BurnConfig, BackgroundTaskConfig,
                  ScriptConfig, UpdateBackgroundServiceConfig]


class WorkloadBuilder():
    def __init__(self):
        self.actions = []

    def add_action(self, action: Action):
        self.actions.append(action)

    def as_payload(self) -> str:
        return json.dumps(Workload(actions=self.actions).model_dump_json())

    def http_request(self, service: str, builder: 'WorkloadBuilder'):
        config = HTTPRequestConfig(
            url=f'http://{service}:8080', body=Workload(actions=builder.actions))
        self.add_action(Action(name=ActionEnum.HTTPRequest, config=config))

    def allocate_memory(self, size_bytes: int, num_allocations: int = 1):
        self.add_action(Action(name=ActionEnum.AllocateMemory, config=AllocateMemoryConfig(
            size_bytes=size_bytes, num_allocations=num_allocations)))

    def burn_cpu(self, duration_ms: int = 1):
        self.add_action(
            Action(name=ActionEnum.Burn, config=BurnConfig(duration=f'{duration_ms}ms')))

    def script(self, script: str):
        self.add_action(Action(name=ActionEnum.Script,
                        config=ScriptConfig(script=script)))

    def redis_get(self, address: str, key: str):
        self.add_action(Action(name=ActionEnum.RedisCommand,
                               config=RedisCommandConfig(command='GET', args=[key], address=address)))

    def redis_set(self, address: str, key: str, value: str):
        self.add_action(Action(name=ActionEnum.RedisCommand,
                               config=RedisCommandConfig(command='SET', args=[key, value], address=address)))

    def postgres_query(self, host: str, query: str, port: int = 5432, dbname: str = "postgres", user: str = "postgres", password: str = "postgres", repeat: int = 1, maxopen: int = 5, maxidle: int = 10):
        self.add_action(Action(name=ActionEnum.PostgresqlQuery, config=PostgresqlQueryConfig(host=host, port=port,
                                                                                             dbname=dbname, user=user, password=password, query=query, repeat=repeat, maxopen=maxopen, maxidle=maxidle)))


class Worker(BaseModel):
    instances: int
    duration: str
    delay: str


class Client(BaseModel):
    workers: List[Worker]


class Workload(BaseModel):
    actions: List[Action]


class Phase(BaseModel):
    name: str
    client: Client
    setup: Optional[Workload] = None
    workload: Optional[Workload] = None


class Plan(BaseModel):
    phases: List[Phase]


def create_auth_check() -> WorkloadBuilder:
    auth = WorkloadBuilder()
    auth.script(script=f'''function run() {{
        // Burn
        ctx.burn("10ms");

        auth_cache = ctx.get_service("auth-cache");
        auth_cache.set(ctx.ctx, "key", "value");
        auth_cache.get(ctx.ctx, "key");
    }}''')
    return auth


app = typer.Typer()

create_upload_app = typer.Typer()
app.add_typer(create_upload_app, name="upload")

show_recommendations_app = typer.Typer()
app.add_typer(show_recommendations_app, name="show-recommendations")

show_video_app = typer.Typer()
app.add_typer(show_video_app, name="show-video")


@show_video_app.command()
def create():
    # Recommendation just looks up the most recent videos
    inventory = WorkloadBuilder()
    inventory.script(script=f'''function run() {{
        ctx.burn("10ms");

        inventory_db = ctx.get_service("inventory-db");
        inventory_db.query(ctx.ctx, "CREATE TABLE IF NOT EXISTS videos (id text PRIMARY KEY, status text, created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP);");
        videos = inventory_db.query(ctx.ctx, "SELECT * FROM videos ORDER BY created_at DESC LIMIT 1;");

        storage = ctx.get_service("storage");
        vid = storage.get_object(ctx.ctx, "videos", videos[0].id);

        ctx.print("Got video: " + videos[0].id + " with size: " + vid.length + " bytes");
    }}''')

    # Frontend receives the video from the user
    frontend = WorkloadBuilder()
    frontend.script(script=f'''function run() {{
        // Send a request for authentication
        ctx.http_post(ctx.ctx, "http://auth:8080", {create_auth_check().as_payload()});

        ctx.http_post(ctx.ctx, "http://inventory:8080", {inventory.as_payload()});
    }}''')

    phase = Phase(name='phase1', client=Client(workers=[Worker(
        instances=1, duration='1200h', delay='2ms')]), workload=Workload(actions=frontend.actions), setup=Workload(actions=[]))
    plan = Plan(phases=[phase])

    yaml.safe_dump(plan.model_dump(), sys.stdout)


@show_recommendations_app.command()
def create():
    # Frontend calls auth service to verify the user
    auth = create_auth_check()

    # Recommendation just looks up the most recent videos
    inventory = WorkloadBuilder()
    inventory.script(script=f'''function run() {{
        ctx.burn("10ms");

        var inventory_db = ctx.get_service("inventory-db");
        var videos = inventory_db.query(ctx.ctx, "SELECT * FROM videos ORDER BY created_at DESC LIMIT 20;");

        // now get all the thumbnails
        var size = 0;
        storage = ctx.get_service("storage");
        for (var i = 0; i < videos.length; i++) {{
            var obj = storage.get_object(ctx.ctx, "thumbnails", videos[i].id);
            size += obj.length;
        }}

        ctx.print("Got " + videos.length + " videos with total size: " + size + " bytes");
    }}''')

    # Frontend calls recommendation service to get top 20 videos
    recommendation = WorkloadBuilder()
    recommendation.script(script=f'''function run() {{
        // Send a request for authentication
        ctx.http_post(ctx.ctx, "http://inventory:8080", {inventory.as_payload()});
    }}''')

    # Frontend receives the video from the user
    frontend = WorkloadBuilder()
    frontend.script(script=f'''function run() {{
        // Send a request for authentication
        ctx.http_post(ctx.ctx, "http://auth:8080", {auth.as_payload()});

        ctx.http_post(ctx.ctx, "http://recommendation:8080", {recommendation.as_payload()});
    }}''')

    phase = Phase(name='phase1', client=Client(workers=[Worker(
        instances=1, duration='1200h', delay='2ms')]), workload=Workload(actions=frontend.actions), setup=Workload(actions=[]))
    plan = Plan(phases=[phase])

    yaml.safe_dump(plan.model_dump(), sys.stdout)


@create_upload_app.command()
def create(
    video_size_kb: Annotated[int, typer.Option(
        help="Size of the video to be uploaded")] = 500,
):
    video_size_bytes = video_size_kb * 1024

    # Frontend calls auth service to verify the user
    auth = create_auth_check()

    # Frontend calls upload service to upload the video
    upload = WorkloadBuilder()
    upload.script(script=f'''function run() {{
        ctx.allocate_memory({video_size_bytes * 2}, 1);
        ctx.burn("100ms");

        var video_data = ctx.random_string({video_size_bytes});

        // Create a unique ID for the video
        video_id = ctx.uuid();

        // Track the video status in the database
        upload_db = ctx.get_service("upload-db");
        upload_db.query(ctx.ctx, "CREATE TABLE IF NOT EXISTS videos (id text PRIMARY KEY, status text);");

        upload_db.query(ctx.ctx, "INSERT INTO videos (id, status) VALUES ('" + video_id + "','uploaded');");

        // Send a message to the data broker
        data_broker = ctx.get_service("data-broker-producer");

        msg = JSON.stringify({{
            "video_id": video_id,
            "raw_data": video_data,
            "status": "uploaded"
        }});

        data_broker.produce(ctx.ctx, "test1", msg);
    }}''')

    # Frontend receives the video from the user
    frontend = WorkloadBuilder()
    # TODO: the memory should be keept until the video is uploaded
    frontend.script(script=f'''function run() {{
        // Allocate memory for the video
        ctx.allocate_memory({video_size_bytes * 2}, 1)

        // Burn CPU to simulate video processing
        ctx.burn("100ms")

        // Send a request for authentication
        ctx.http_post(ctx.ctx, "http://auth:8080", {auth.as_payload()});

        // TODO: Send custom payload
        // Send a request to upload the video
        ctx.http_post(ctx.ctx, "http://upload:8080", {upload.as_payload()});
    }}''')

    phase = Phase(name='phase1', client=Client(workers=[Worker(
        instances=1, duration='1200h', delay='1s')]), workload=Workload(actions=frontend.actions), setup=Workload(actions=[]))
    plan = Plan(phases=[phase])

    yaml.safe_dump(plan.model_dump(), sys.stdout)


if __name__ == '__main__':
    app()
